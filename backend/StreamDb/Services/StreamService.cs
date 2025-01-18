using Grpc.Core;
using Microsoft.EntityFrameworkCore;
using StreamDb.Context;
using StreamDb.Models;
using StreamDb.Protos;
using Google.Protobuf.WellKnownTypes;

namespace StreamDb.Services;

public class StreamService(StreamDbContext context) : Protos.StreamService.StreamServiceBase
{
    public override async Task<StreamResponse> CreateStream(CreateStreamRequest request, ServerCallContext context1)
    {
        // Validate request
        if (string.IsNullOrWhiteSpace(request.Title))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Title is required"));
        }

        if (string.IsNullOrWhiteSpace(request.StreamKey))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Stream key is required"));
        }

        // Validate user exists and is not deleted
        var user = await context.Users
            .FirstOrDefaultAsync(u => u.Id == request.UserId);

        if (user == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));
        }

        if (user.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Cannot create stream for deleted user"));
        }

        // Check for duplicate stream key
        var streamKeyExists = await context.Streams
            .AnyAsync(s => s.StreamKey == request.StreamKey && s.DeletedAt == null);

        if (streamKeyExists)
        {
            throw new RpcException(new Status(StatusCode.AlreadyExists, "Stream key already in use"));
        }

        // Validate time range
        var startTime = request.StartTime;
        var endTime = request.EndTime;

        var stream = new Streams
        {
            Title = request.Title.Trim(),
            Description = request.Description.Trim(),
            StartTime = startTime,
            EndTime = endTime,
            StreamKey = request.StreamKey.Trim(),
            Resolution = request.Resolution.Trim(),
            Bitrate = request.Bitrate.Trim(),
            Framerate = request.Framerate.Trim(),
            Codec = request.Codec.Trim(),
            Protocol = request.Protocol,
            Status = (EStreamStatus)request.Status,
            UserId = request.UserId,
            ViewCount = 0
        };

        try
        {
            context.Streams.Add(stream);
            await context.SaveChangesAsync();
            return CreateStreamResponse(stream);
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to create stream"));
        }
    }

    public override async Task<StreamResponse> GetStream(GetStreamRequest request, ServerCallContext context1)
    {
        var stream = await context.Streams
            .AsNoTracking()
            .Include(s => s.User)
            .FirstOrDefaultAsync(s => s.Id == request.Id);

        if (stream == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Stream not found"));
        }

        if (stream.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Stream has been deleted"));
        }

        if (stream.User.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Stream's user has been deleted"));
        }

        return CreateStreamResponse(stream);
    }

    public override async Task<StreamResponse> UpdateStream(UpdateStreamRequest request, ServerCallContext context1)
    {
        var stream = await context.Streams
            .Include(s => s.User)
            .FirstOrDefaultAsync(s => s.Id == request.Id);

        if (stream == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Stream not found"));
        }

        if (stream.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Cannot update deleted stream"));
        }

        if (stream.User.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Cannot update stream of deleted user"));
        }

        // Validate time range if updating times
        var startTime = request.StartTime;
        var endTime = request.EndTime;

        stream.Title = request.Title?.Trim() ?? stream.Title;
        stream.Description = request.Description.Trim();
        stream.StartTime = startTime;
        stream.EndTime = endTime;
        stream.Resolution = request.Resolution?.Trim() ?? stream.Resolution;
        stream.Bitrate = request.Bitrate?.Trim() ?? stream.Bitrate;
        stream.Framerate = request.Framerate?.Trim() ?? stream.Framerate;
        stream.Codec = request.Codec?.Trim() ?? stream.Codec;
        stream.Protocol = request.Protocol;

        try
        {
            await context.SaveChangesAsync();
            return CreateStreamResponse(stream);
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to update stream"));
        }
    }

    public override async Task<Empty> DeleteStream(DeleteStreamRequest request, ServerCallContext context1)
    {
        var stream = await context.Streams
            .FirstOrDefaultAsync(s => s.Id == request.Id);

        if (stream == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Stream not found"));
        }

        if (stream.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Stream is already deleted"));
        }

        if (stream.Status == EStreamStatus.ONLINE)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                "Cannot delete an active stream. Please end the stream first."));
        }

        try
        {
            stream.DeletedAt = DateTime.UtcNow;
            await context.SaveChangesAsync();
            return new Empty();
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to delete stream"));
        }
    }

    public override async Task<ListStreamsResponse> ListStreams(ListStreamsRequest request, ServerCallContext context1)
    {
        try
        {
            var query = context.Streams
                .AsNoTracking()
                .Include(s => s.User)
                .Where(s => s.DeletedAt == null && s.User.DeletedAt == null);

            var streams = await query
                .OrderByDescending(s => s.CreatedAt)
                .Select(s => CreateStreamResponse(s))
                .ToListAsync();

            var response = new ListStreamsResponse();
            response.Streams.AddRange(streams);
            return response;
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to retrieve streams"));
        }
    }

    private static StreamResponse CreateStreamResponse(Streams stream)
    {
        return new StreamResponse
        {
            Id = stream.Id,
            Title = stream.Title,
            Description = stream.Description,
            StartTime = stream.StartTime,
            EndTime = stream.EndTime,
            StreamKey = stream.StreamKey,
            Resolution = stream.Resolution,
            Bitrate = stream.Bitrate,
            Framerate = stream.Framerate,
            Codec = stream.Codec,
            ViewCount = stream.ViewCount,
            Protocol = stream.Protocol,
            Status = (StreamStatus)stream.Status,
            UserId = stream.UserId,
        };
    }

    private static void ValidateStatusTransition(EStreamStatus currentStatus, EStreamStatus newStatus)
    {
        var isValidTransition = (currentStatus, newStatus) switch
        {
            (EStreamStatus.SCHEDULED, EStreamStatus.ONLINE) => true,
            (EStreamStatus.SCHEDULED, EStreamStatus.OFFLINE) => true,
            (EStreamStatus.ONLINE, EStreamStatus.OFFLINE) => true,
            (EStreamStatus.ONLINE, EStreamStatus.COMPLETE) => true,
            (EStreamStatus.OFFLINE, EStreamStatus.ONLINE) => true,
            (EStreamStatus.OFFLINE, EStreamStatus.COMPLETE) => true,
            _ => false
        };

        if (!isValidTransition)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                $"Invalid status transition from {currentStatus} to {newStatus}"));
        }
    }
}