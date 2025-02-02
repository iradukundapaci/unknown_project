using Grpc.Core;
using Microsoft.EntityFrameworkCore;
using StreamDb.Context;
using StreamDb.Models;
using StreamDb.Protos;
using Google.Protobuf.WellKnownTypes;
using System.Linq.Expressions;

namespace StreamDb.Services;

public class UserService(StreamDbContext context) : Protos.UserService.UserServiceBase
{
    private const int MaxPageSize = 10;

    public override async Task<UserResponse> CreateUser(CreateUserRequest request, ServerCallContext context1)
    {
        ValidateCreateRequest(request);

        var existingUser = await context.Users
            .FirstOrDefaultAsync(u => u.Email == request.Email || u.ClerkId == request.ClerkId);

        if (existingUser != null)
        {
            if (existingUser.DeletedAt == null)
                throw new RpcException(new Status(StatusCode.AlreadyExists,
                    "User with this email or ClerkId already exists"));
            existingUser.DeletedAt = null;
            existingUser.FirstName = request.FirstName;
            existingUser.LastName = request.LastName;
            existingUser.ProfileImageUrl = request.ProfileImageUrl;
            await context.SaveChangesAsync();

            return CreateUserResponse(existingUser);

        }
        
        var user = new User
        {
            Email = request.Email.Trim().ToLower(),
            FirstName = request.FirstName.Trim(),
            LastName = request.LastName.Trim(),
            ProfileImageUrl = request.ProfileImageUrl.Trim(),
            ClerkId = request.ClerkId.Trim()
        };

        try
        {
            context.Users.Add(user);
            await context.SaveChangesAsync();
            return CreateUserResponse(user);
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to create user: {ex.Message}"));
        }
    }

    public override async Task<UserResponse> GetUser(GetUserRequest request, ServerCallContext context1)
    {
        var user = await context.Users
            .AsNoTracking()
            .FirstOrDefaultAsync(u => u.Id == request.Id || u.ClerkId == request.ClerkId);

        if (user is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));
        }

        return CreateUserResponse(user);
    }

    public override async Task<UserResponse> UpdateUser(UpdateUserRequest request, ServerCallContext context1)
    {
        ValidateUpdateRequest(request);

        var user = await context.Users
            .FirstOrDefaultAsync(u => u.Id == request.Id);

        if (user is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));
        }

        await ValidateEmailUniqueness(request.Email, request.Id);
        
        UpdateUserFields(user, request);

        try
        {
            await context.SaveChangesAsync();
            return CreateUserResponse(user);
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to update user: {ex.Message}"));
        }
    }

    public override async Task<Empty> DeleteUser(DeleteUserRequest request, ServerCallContext context1)
    {
        var user = await context.Users
            .FirstOrDefaultAsync(u => u.Id == request.Id);

        if (user is not { DeletedAt: null })
        {
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));
        }

        await ValidateUserDeletion(user);

        try
        {
            user.DeletedAt = DateTime.UtcNow;
            await context.SaveChangesAsync();
            return new Empty();
        }
        catch (Exception ex)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to delete user: {ex.Message}"));
        }
    }

    public override async Task<ListUsersResponse> ListUsers(ListUsersRequest request, ServerCallContext context1)
    {
        try
        {
            var query = context.Users
                .AsNoTracking()
                .Where(u => u.DeletedAt == null);

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

            var users = await query
                .Skip((pageNumber - 1) * pageSize)
                .Take(pageSize)
                .ToListAsync();

            return new ListUsersResponse
            {
                Users = { users.Select(CreateUserResponse) },
                Pagination = new PaginationMetadata
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
            throw new RpcException(new Status(StatusCode.Internal, $"Failed to retrieve users: {ex.Message}"));
        }
    }

    #region Validation Methods

    private static void ValidateCreateRequest(CreateUserRequest request)
    {
        var errors = new List<string>();

        if (string.IsNullOrWhiteSpace(request.Email))
            errors.Add("Email is required");
        else if (!IsValidEmail(request.Email))
            errors.Add("Invalid email format");

        if (string.IsNullOrWhiteSpace(request.FirstName))
            errors.Add("First name is required");

        if (string.IsNullOrWhiteSpace(request.LastName))
            errors.Add("Last name is required");

        if (string.IsNullOrWhiteSpace(request.ProfileImageUrl))
            errors.Add("Profile image URL is required");

        if (string.IsNullOrWhiteSpace(request.ClerkId))
            errors.Add("ClerkId is required");

        if (errors.Count != 0)
            throw new RpcException(new Status(StatusCode.InvalidArgument, string.Join(", ", errors)));
    }

    private static void ValidateUpdateRequest(UpdateUserRequest request)
    {
        if (request.Id <= 0)
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Invalid user ID"));

        var errors = new List<string>();

        if (!string.IsNullOrWhiteSpace(request.Email) && !IsValidEmail(request.Email))
            errors.Add("Invalid email format");

        if (errors.Count != 0)
            throw new RpcException(new Status(StatusCode.InvalidArgument, string.Join(", ", errors)));
    }

    private async Task ValidateEmailUniqueness(string email, int userId)
    {
        if (string.IsNullOrWhiteSpace(email)) return;

        var normalizedEmail = email.Trim().ToLower();
        var emailExists = await context.Users
            .AnyAsync(u => u.Email == normalizedEmail && u.Id != userId && u.DeletedAt == null);

        if (emailExists)
            throw new RpcException(new Status(StatusCode.AlreadyExists, "Email already in use"));
    }

    private async Task ValidateUserDeletion(User user)
    {
        var hasActiveStreams = await context.Streams
            .AnyAsync(s => s.UserId == user.Id && 
                          s.DeletedAt == null && 
                          (s.Status == EStreamStatus.ONLINE || s.Status == EStreamStatus.SCHEDULED));

        if (hasActiveStreams)
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                "Cannot delete user with active streams. Please end or cancel all streams first."));
    }

    private static bool IsValidEmail(string email)
    {
        try
        {
            var addr = new System.Net.Mail.MailAddress(email);
            return addr.Address == email;
        }
        catch
        {
            return false;
        }
    }

    #endregion

    #region Helper Methods

    private static UserResponse CreateUserResponse(User user)
    {
        return new UserResponse
        {
            Id = user.Id,
            Email = user.Email,
            FirstName = user.FirstName,
            LastName = user.LastName,
            ProfileImageUrl = user.ProfileImageUrl,
            ClerkId = user.ClerkId
        };
    }

    private static void UpdateUserFields(User user, UpdateUserRequest request)
    {
        if (!string.IsNullOrWhiteSpace(request.Email))
            user.Email = request.Email.Trim().ToLower();
        
        if (!string.IsNullOrWhiteSpace(request.FirstName))
            user.FirstName = request.FirstName.Trim();
        
        if (!string.IsNullOrWhiteSpace(request.LastName))
            user.LastName = request.LastName.Trim();

        if (!string.IsNullOrWhiteSpace(request.ProfileImageUrl))
            user.ProfileImageUrl = request.ProfileImageUrl.Trim();
    }

    private static IQueryable<User> ApplyFilters(IQueryable<User> query, UserFilter? filter)
    {
        if (filter == null) return query;

        if (!string.IsNullOrWhiteSpace(filter.EmailContains))
            query = query.Where(u => u.Email.Contains(filter.EmailContains));

        if (!string.IsNullOrWhiteSpace(filter.NameContains))
            query = query.Where(u => u.FirstName.Contains(filter.NameContains) || 
                                   u.LastName.Contains(filter.NameContains));

        if (filter.IdEquals > 0)
            query = query.Where(u => u.Id == filter.IdEquals);

        return query;
    }

    private static IQueryable<User> ApplySorting(IQueryable<User> query, string? sortBy, bool ascending)
    {
        Expression<Func<User, object>> keySelector = sortBy?.ToLower() switch
        {
            "email" => user => user.Email,
            "firstname" => user => user.FirstName,
            "lastname" => user => user.LastName,
            "id" => user => user.Id,
            _ => user => user.Id
        };

        return ascending ? query.OrderBy(keySelector) : query.OrderByDescending(keySelector);
    }

    #endregion
}