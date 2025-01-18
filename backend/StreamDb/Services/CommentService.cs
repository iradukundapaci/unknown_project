using Grpc.Core;
using Microsoft.EntityFrameworkCore;
using StreamDb.Context;
using StreamDb.Models;
using StreamDb.Protos;
using Google.Protobuf.WellKnownTypes;

namespace StreamDb.Services;

public class CommentService(StreamDbContext context) : Protos.CommentService.CommentServiceBase
{
    private const int MaxCommentLength = 1000; // Maximum length for comments

    public override async Task<CommentResponse> CreateComment(CreateCommentRequest request, ServerCallContext context1)
    {
        // Validate comment content
        if (string.IsNullOrWhiteSpace(request.Message))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Comment message is required"));
        }

        if (request.Message.Length > MaxCommentLength)
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, 
                $"Comment exceeds maximum length of {MaxCommentLength} characters"));
        }

        // Validate user exists and is not deleted
        var user = await context.Users
            .AsNoTracking()
            .FirstOrDefaultAsync(u => u.Id == request.UserId);

        if (user == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));
        }

        if (user.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Cannot create comment for deleted user"));
        }

        // Validate stream exists and is not deleted
        var stream = await context.Streams
            .AsNoTracking()
            .FirstOrDefaultAsync(s => s.Id == request.StreamId);

        if (stream == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Stream not found"));
        }

        if (stream.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Cannot comment on deleted stream"));
        }

        // Validate stream status allows comments
        if (stream.Status == EStreamStatus.COMPLETE)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                "Cannot comment on completed streams"));
        }

        var comment = new Comments
        {
            Message = request.Message.Trim(),
            UserId = request.UserId,
            StreamId = request.StreamId
        };

        try
        {
            context.Comments.Add(comment);
            await context.SaveChangesAsync();
            return CreateCommentResponseWithDetails(comment);
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to create comment"));
        }
    }

    public override async Task<CommentResponse> GetComment(GetCommentRequest request, ServerCallContext context1)
    {
        var comment = await context.Comments
            .AsNoTracking()
            .Include(c => c.User)
            .Include(c => c.Stream)
            .FirstOrDefaultAsync(c => c.Id == request.Id);

        if (comment == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Comment not found"));
        }

        if (comment.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Comment has been deleted"));
        }

        if (comment.User.DeletedAt != null || comment.Stream.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                "Comment's user or stream has been deleted"));
        }

        return CreateCommentResponseWithDetails(comment);
    }

    public override async Task<CommentResponse> UpdateComment(UpdateCommentRequest request, ServerCallContext context1)
    {
        var comment = await context.Comments
            .Include(c => c.User)
            .Include(c => c.Stream)
            .FirstOrDefaultAsync(c => c.Id == request.Id);

        if (comment == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Comment not found"));
        }

        if (comment.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Cannot update deleted comment"));
        }

        if (comment.User.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                "Cannot update comment of deleted user"));
        }

        if (comment.Stream.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                "Cannot update comment on deleted stream"));
        }

        // Validate new message
        if (string.IsNullOrWhiteSpace(request.Message))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Comment message is required"));
        }

        if (request.Message.Length > MaxCommentLength)
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, 
                $"Comment exceeds maximum length of {MaxCommentLength} characters"));
        }

        try
        {
            comment.Message = request.Message.Trim();
            await context.SaveChangesAsync();
            return CreateCommentResponseWithDetails(comment);
        }
        catch 
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to update comment"));
        }
    }

    public override async Task<Empty> DeleteComment(DeleteCommentRequest request, ServerCallContext context1)
    {
        var comment = await context.Comments
            .Include(c => c.Stream)
            .FirstOrDefaultAsync(c => c.Id == request.Id);

        if (comment == null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Comment not found"));
        }

        if (comment.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Comment is already deleted"));
        }

        try
        {
            comment.DeletedAt = DateTime.UtcNow;
            await context.SaveChangesAsync();
            return new Empty();
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to delete comment"));
        }
    }

    public override async Task<ListCommentsResponse> ListComments(ListCommentsRequest request, ServerCallContext context1)
    {
        try
        {
            var query = context.Comments
                .AsNoTracking()
                .Include(c => c.User)
                .Include(c => c.Stream)
                .Where(c => c.DeletedAt == null &&
                            c.User.DeletedAt == null &&
                            c.Stream.DeletedAt == null);

            var comments = await query
                .OrderByDescending(c => c.CreatedAt)
                .Select(c => CreateCommentResponseWithDetails(c))
                .ToListAsync();

            var response = new ListCommentsResponse();
            response.Comments.AddRange(comments);
            return response;
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to retrieve comments"));
        }
    }

    private static CommentResponse CreateCommentResponseWithDetails(Comments comment)
    {
        return new CommentResponse
        {
            Id = comment.Id,
            Message = comment.Message,
            UserId = comment.UserId,
            StreamId = comment.StreamId,
        };
    }
}