using System.Globalization;
using Grpc.Core;
using Microsoft.EntityFrameworkCore;
using StreamDb.Context;
using StreamDb.Models;
using StreamDb.Protos;
using Google.Protobuf.WellKnownTypes;
using System.Linq.Expressions;

namespace StreamDb.Services;

public class StreamService(StreamDbContext context) : Protos.StreamService.StreamServiceBase
{
    private const int MaxPageSize = 10;
    private const string TimeFormat = "yyyy-MM-ddTHH:mm:ssZ";

    public override async Task<StreamResponse> CreateStream(CreateStreamRequest request, ServerCallContext context1)
    {
        ValidateCreateRequest(request);
        await ValidateUserExists(request.UserId);

        var stream = new Streams()
        {
            Title = request.Title.Trim(),
            Description = request.Description?.Trim()!,
            StartTime = ParseTimestamp(request.StartTime),
            EndTime = ParseTimestamp(request.EndTime),
            StreamKey = request.StreamKey.Trim(),
            Resolution = request.Resolution?.Trim()!,
            Bitrate = request.Bitrate,
            Framerate = request.Framerate,
            Codec = request.Codec?.Trim()!,
            Protocol = request.Protocol?.Trim()!,
            Status = ConvertStreamStatus(request.Status),
            UserId = (int)request.UserId,
            ViewCount = 0,
            CreatedAt = DateTime.UtcNow
        };

        try
        {
            context.Streams.Add(stream);
            await context.SaveChangesAsync();
            return CreateStreamResponse(stream);
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to create stream: {ex.Message}"));
        }
    }

    public override async Task<StreamResponse> GetStream(GetStreamRequest request, ServerCallContext context1)
    {
        var stream = await context.Streams
            .AsNoTracking()
            .FirstOrDefaultAsync(s => s.Id == request.Id);

        if (stream is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Stream not found"));
        }

        return CreateStreamResponse(stream);
    }

    public override async Task<StreamResponse> UpdateStream(UpdateStreamRequest request, ServerCallContext context1)
    {
        ValidateUpdateRequest(request);

        var stream = await context.Streams
            .FirstOrDefaultAsync(s => s.Id == request.Id);

        if (stream is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Stream not found"));
        }

        UpdateStreamFields(stream, request);

        try
        {
            await context.SaveChangesAsync();
            return CreateStreamResponse(stream);
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to update stream: {ex.Message}"));
        }
    }

    public override async Task<Empty> DeleteStream(DeleteStreamRequest request, ServerCallContext context1)
    {
        var stream = await context.Streams
            .FirstOrDefaultAsync(s => s.Id == request.Id);

        if (stream is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Stream not found"));
        }

        ValidateStreamDeletion(stream);

        try
        {
            stream.DeletedAt = DateTime.UtcNow;
            await context.SaveChangesAsync();
            return new Empty();
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to delete stream: {ex.Message}"));
        }
    }

public override async Task<ListStreamsResponse> ListStreams(ListStreamsRequest request, ServerCallContext context1)
    {
        try
        {
            var query = context.Streams
                .AsNoTracking()
                .Include(s => s.User)
                .Where(s => s.DeletedAt == null);

            // Apply filters
            query = ApplyFilters(query, request.Filter);

            // Apply sorting
            query = ApplySorting(query, request.SortBy, request.Ascending);

            // Get total count for pagination
            var totalItems = await query.CountAsync();

            // Handle pagination parameters
            var pageSize = request.PageSize <= 0 ? MaxPageSize : Math.Min(request.PageSize, MaxPageSize);
            var pageNumber = request.PageNumber <= 0 ? 1 : request.PageNumber;
            var totalPages = (int)Math.Ceiling(totalItems / (double)pageSize);

            // Ensure totalPages is at least 1 when there are items
            totalPages = totalPages <= 0 && totalItems > 0 ? 1 : totalPages;

            // If pageNumber is greater than totalPages, set it to the last page
            if (totalPages > 0 && pageNumber > totalPages)
            {
                pageNumber = totalPages;
            }

            var streams = await query
                .Skip((pageNumber - 1) * pageSize)
                .Take(pageSize)
                .ToListAsync();

            return new ListStreamsResponse
            {
                Streams = { streams.Select(CreateStreamResponse) },
                MetaData = new PaginationMetadata
                {
                    TotalItems = totalItems,
                    TotalPages = totalPages,
                    CurrentPage = pageNumber,
                    PageSize = pageSize
                }
            };
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to retrieve streams: {ex.Message}"));
        }
    }


    private static void ValidateCreateRequest(CreateStreamRequest request)
    {
        var errors = new List<string>();

        if (string.IsNullOrWhiteSpace(request.Title))
            errors.Add("Title is required");

        if (string.IsNullOrWhiteSpace(request.StreamKey))
            errors.Add("Stream key is required");

        try
        {
            var startTime = ParseTimestamp(request.StartTime);
            var endTime = ParseTimestamp(request.EndTime);

            var now = DateTime.UtcNow;
            
            if (startTime <= now)
            {
                errors.Add("Start time must be in the future");
            }

            if (startTime >= endTime)
            {
                errors.Add("Start time must be before end time");
            }
        }
        catch (RpcException ex) when (ex.StatusCode == StatusCode.InvalidArgument)
        {
            errors.Add(ex.Status.Detail);
        }

        if (request.UserId <= 0)
            errors.Add("Invalid user ID");

        if (request.Bitrate <= 0)
            errors.Add("Bitrate must be greater than 0");

        if (request.Framerate <= 0)
            errors.Add("Framerate must be greater than 0");

        if (!string.IsNullOrWhiteSpace(request.Resolution))
        {
            if (!IsValidResolution(request.Resolution))
            {
                errors.Add("Invalid resolution format. Expected format: WidthxHeight");
            }
        }

        if (errors.Count != 0)
            throw new RpcException(new Status(StatusCode.InvalidArgument, string.Join(", ", errors)));
    }
    
    private static DateTime ParseTimestamp(string timestamp)
    {
        if (!DateTime.TryParseExact(timestamp, TimeFormat, CultureInfo.InvariantCulture, 
                DateTimeStyles.AdjustToUniversal, out DateTime parsedTime))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, 
                $"Invalid timestamp format. Expected format: {TimeFormat}"));
        }
        return parsedTime;
    }
    
    private static bool IsValidResolution(string resolution)
    {
        if (string.IsNullOrWhiteSpace(resolution)) return false;
        
        var parts = resolution.Split('x');
        if (parts.Length != 2) return false;

        return int.TryParse(parts[0], out int width) && 
               int.TryParse(parts[1], out int height) && 
               width > 0 && height > 0;
    }

    private static void ValidateUpdateRequest(UpdateStreamRequest request)
    {
        var errors = new List<string>();

        if (string.IsNullOrWhiteSpace(request.Title))
            errors.Add("Title is required");

        try
        {
            var startTime = ParseTimestamp(request.StartTime);
            var endTime = ParseTimestamp(request.EndTime);

            var now = DateTime.UtcNow;
            
            if (startTime <= now)
            {
                errors.Add("Start time must be in the future");
            }

            if (startTime >= endTime)
            {
                errors.Add("Start time must be before end time");
            }
        }
        catch (RpcException ex) when (ex.StatusCode == StatusCode.InvalidArgument)
        {
            errors.Add(ex.Status.Detail);
        }

        if (request.Bitrate <= 0)
            errors.Add("Bitrate must be greater than 0");

        if (request.Framerate <= 0)
            errors.Add("Framerate must be greater than 0");

        if (!string.IsNullOrWhiteSpace(request.Resolution))
        {
            if (!IsValidResolution(request.Resolution))
            {
                errors.Add("Invalid resolution format. Expected format: WidthxHeight");
            }
        }

        if (errors.Count != 0)
            throw new RpcException(new Status(StatusCode.InvalidArgument, string.Join(", ", errors)));
    }

    private async Task ValidateUserExists(long userId)
    {
        var userExists = await context.Users
            .AnyAsync(u => u.Id == userId && u.DeletedAt == null);

        if (!userExists)
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));
    }

    private static void ValidateStreamDeletion(Streams stream)
    {
        if (stream.Status == EStreamStatus.ONLINE)
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                "Cannot delete an active stream. Please end the stream first."));

        // Add any additional validation logic here
    }

    private static StreamResponse CreateStreamResponse(Streams stream)
    {
        return new StreamResponse
        {
            Id = stream.Id,
            Title = stream.Title,
            Description = stream.Description,
            StartTime = stream.StartTime.ToString(TimeFormat),
            EndTime = stream.EndTime.ToString(TimeFormat),
            StreamKey = stream.StreamKey,
            Resolution = stream.Resolution,
            Bitrate = stream.Bitrate.ToString(),
            Framerate = stream.Framerate.ToString(),
            Codec = stream.Codec,
            ViewCount = stream.ViewCount,
            Protocol = stream.Protocol,
            Status = (StreamStatus)stream.Status,
            UserId = stream.UserId
        };
    }

    private static void UpdateStreamFields(Streams stream, UpdateStreamRequest request)
    {
        if (!string.IsNullOrWhiteSpace(request.Title))
            stream.Title = request.Title.Trim();

        if (request.Description != null)
            stream.Description = request.Description.Trim();

        if (request.StartTime != null)
            stream.StartTime = ParseTimestamp(request.StartTime);

        if (request.EndTime != null)
            stream.EndTime = ParseTimestamp(request.EndTime);

        if (!string.IsNullOrWhiteSpace(request.Resolution))
            stream.Resolution = request.Resolution.Trim();

        if (request.Bitrate > 0)
            stream.Bitrate = request.Bitrate;

        if (request.Framerate > 0)
            stream.Framerate = request.Framerate;

        if (!string.IsNullOrWhiteSpace(request.Codec))
            stream.Codec = request.Codec.Trim();

        if (!string.IsNullOrWhiteSpace(request.Protocol))
            stream.Protocol = request.Protocol;

        if (!string.IsNullOrWhiteSpace(request.Status.ToString()))
            stream.Status = ConvertStreamStatus(request.Status);

        if (request.ViewCount >= 0)
            stream.ViewCount = request.ViewCount;
    }

private static IQueryable<Streams> ApplyFilters(IQueryable<Streams> query, StreamFilter? filter)
    {
        if (filter == null) return query;

        if (!string.IsNullOrWhiteSpace(filter.TitleContains))
            query = query.Where(s => s.Title.Contains(filter.TitleContains));

        if (!string.IsNullOrWhiteSpace(filter.DescriptionContains))
            query = query.Where(s => s.Description.Contains(filter.DescriptionContains));

        if (filter.UserId > 0)
            query = query.Where(s => s.UserId == filter.UserId);

        if (filter.MinViewCount > 0)
            query = query.Where(s => s.ViewCount >= filter.MinViewCount);

        if (filter.MaxViewCount > 0)
            query = query.Where(s => s.ViewCount <= filter.MaxViewCount);

        if (filter.Status.Count > 0)
        {
            var statuses = filter.Status
                .Select(ConvertStreamStatus)
                .Distinct()
                .ToList();
            query = query.Where(s => statuses.Contains(s.Status));
        }

        if (!string.IsNullOrWhiteSpace(filter.Codec))
            query = query.Where(s => s.Codec == filter.Codec);

        if (!string.IsNullOrWhiteSpace(filter.Protocol))
            query = query.Where(s => s.Protocol == filter.Protocol);

        return query;
    }

    private static IQueryable<Streams> ApplySorting(IQueryable<Streams> query, string? sortBy, bool ascending)
    {
        Expression<Func<Streams, object>> keySelector = sortBy?.ToLower() switch
        {
            "title" => stream => stream.Title,
            "startTime" => stream => stream.StartTime,
            "endTime" => stream => stream.EndTime,
            "viewCount" => stream => stream.ViewCount,
            "status" => stream => stream.Status,
            "userid" => stream => stream.UserId,
            _ => stream => stream.Id
        };

        return ascending ? query.OrderBy(keySelector) : query.OrderByDescending(keySelector);
    }
    
    private static EStreamStatus ConvertStreamStatus(StreamStatus status)
    {
        return (EStreamStatus)status;
    }
}