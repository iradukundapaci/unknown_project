using Grpc.Core;
using Microsoft.EntityFrameworkCore;
using StreamDb.Context;
using StreamDb.Models;
using StreamDb.Protos;
using Google.Protobuf.WellKnownTypes;
using System.Linq.Expressions;

namespace StreamDb.Services;

public class CommentService(StreamDbContext context) : Protos.CommentService.CommentServiceBase
{
    private const int MaxPageSize = 10;

    public override async Task<CommentResponse> CreateComment(CreateCommentRequest request, ServerCallContext context1)
    {
        ValidateCreateRequest(request);

        await ValidateRelationships(request.UserId, request.StreamId);

        var comment = new Comments
        {
            Message = request.Message.Trim(),
            UserId = request.UserId,
            StreamId = request.StreamId,
            CreatedAt = DateTime.UtcNow
        };

        try
        {
            context.Comments.Add(comment);
            await context.SaveChangesAsync();
            return CreateCommentResponse(comment);
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to create comment: {ex.Message}"));
        }
    }

    public override async Task<CommentResponse> GetComment(GetCommentRequest request, ServerCallContext context1)
    {
        var comment = await context.Comments
            .AsNoTracking()
            .FirstOrDefaultAsync(c => c.Id == request.Id);

        if (comment is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Comment not found"));
        }

        return CreateCommentResponse(comment);
    }

    public override async Task<CommentResponse> UpdateComment(UpdateCommentRequest request, ServerCallContext context1)
    {
        ValidateUpdateRequest(request);

        var comment = await context.Comments
            .FirstOrDefaultAsync(c => c.Id == request.Id);

        if (comment is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Comment not found"));
        }

        comment.Message = request.Message.Trim();

        try
        {
            await context.SaveChangesAsync();
            return CreateCommentResponse(comment);
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to update comment: {ex.Message}"));
        }
    }

    public override async Task<Empty> DeleteComment(DeleteCommentRequest request, ServerCallContext context1)
    {
        var comment = await context.Comments
            .FirstOrDefaultAsync(c => c.Id == request.Id);

        if (comment is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "Comment not found"));
        }

        try
        {
            comment.DeletedAt = DateTime.UtcNow;
            await context.SaveChangesAsync();
            return new Empty();
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to delete comment: {ex.Message}"));
        }
    }
    public override async Task<ListCommentsResponse> ListComments(ListCommentsRequest request, ServerCallContext context1)
    {
        try
        {
            var query = context.Comments
                .AsNoTracking()
                .Where(c => c.DeletedAt == null);

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

            var comments = await query
                .Skip((pageNumber - 1) * pageSize)
                .Take(pageSize)
                .ToListAsync();

            return new ListCommentsResponse
            {
                Comments = { comments.Select(CreateCommentResponse) },
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
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to retrieve comments: {ex.Message}"));
        }
    }
    #region Validation Methods

    private static void ValidateCreateRequest(CreateCommentRequest request)
    {
        var errors = new List<string>();

        if (string.IsNullOrWhiteSpace(request.Message))
            errors.Add("Message is required");

        if (request.UserId <= 0)
            errors.Add("Invalid user ID");

        if (request.StreamId <= 0)
            errors.Add("Invalid stream ID");

        if (errors.Count > 0)
            throw new RpcException(new Status(StatusCode.InvalidArgument, string.Join(", ", errors)));
    }

    private static void ValidateUpdateRequest(UpdateCommentRequest request)
    {
        var errors = new List<string>();

        if (request.Id <= 0)
            errors.Add("Invalid comment ID");

        if (string.IsNullOrWhiteSpace(request.Message))
            errors.Add("Message is required");

        if (errors.Count > 0)
            throw new RpcException(new Status(StatusCode.InvalidArgument, string.Join(", ", errors)));
    }

    private async Task ValidateRelationships(int userId, int streamId)
    {
        var user = await context.Users
            .AsNoTracking()
            .FirstOrDefaultAsync(u => u.Id == userId && u.DeletedAt == null);

        if (user == null)
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));

        var stream = await context.Streams
            .AsNoTracking()
            .FirstOrDefaultAsync(s => s.Id == streamId && s.DeletedAt == null);

        if (stream == null)
            throw new RpcException(new Status(StatusCode.NotFound, "Stream not found"));
    }

    #endregion

    #region Helper Methods

    private static CommentResponse CreateCommentResponse(Comments comment)
    {
        return new CommentResponse
        {
            Id = comment.Id,
            Message = comment.Message,
            UserId = comment.UserId,
            StreamId = comment.StreamId,
            CreatedAt = comment.CreatedAt.ToString("O")
        };
    }

    private static IQueryable<Comments> ApplyFilters(IQueryable<Comments> query, CommentFilter? filter)
    {
        if (filter == null) return query;

        if (!string.IsNullOrWhiteSpace(filter.MessageContains))
            query = query.Where(c => c.Message.Contains(filter.MessageContains));

        if (filter.UserId > 0)
            query = query.Where(c => c.UserId == filter.UserId);

        if (filter.StreamId > 0)
            query = query.Where(c => c.StreamId == filter.StreamId);

        if (!string.IsNullOrWhiteSpace(filter.CreatedAfter) && 
            DateTime.TryParse(filter.CreatedAfter, out var createdAfter))
            query = query.Where(c => c.CreatedAt >= createdAfter);

        if (!string.IsNullOrWhiteSpace(filter.CreatedBefore) && 
            DateTime.TryParse(filter.CreatedBefore, out var createdBefore))
            query = query.Where(c => c.CreatedAt <= createdBefore);

        return query;
    }

    private static IQueryable<Comments> ApplySorting(IQueryable<Comments> query, string? sortBy, bool ascending)
    {
        Expression<Func<Comments, object>> keySelector = sortBy?.ToLower() switch
        {
            "message" => comment => comment.Message,
            "userId" => comment => comment.UserId,
            "streamId" => comment => comment.StreamId,
            "createdAt" => comment => comment.CreatedAt,
            _ => comment => comment.Id
        };

        return ascending ? query.OrderBy(keySelector) : query.OrderByDescending(keySelector);
    }

    #endregion
}