using Grpc.Core;
using Microsoft.EntityFrameworkCore;
using StreamDb.Context;
using StreamDb.Models;
using StreamDb.Protos;
using Google.Protobuf.WellKnownTypes;

namespace StreamDb.Services;

public class UserService(StreamDbContext context) : Protos.UserService.UserServiceBase
{
    public override async Task<UserResponse> CreateUser(CreateUserRequest request, ServerCallContext context1)
    {
        if (string.IsNullOrWhiteSpace(request.Email))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Email is required"));
        }

        if (string.IsNullOrWhiteSpace(request.Names))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Names are required"));
        }

        if (!IsValidEmail(request.Email))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Invalid email format"));
        }
        
        var existingUser = await context.Users
            .FirstOrDefaultAsync(u => u.Email == request.Email);

        if (existingUser != null)
        {
            if (existingUser.DeletedAt != null)
            {
                existingUser.DeletedAt = null;
                existingUser.Names = request.Names;
                await context.SaveChangesAsync();

                return CreateUserResponse(existingUser);
            }

            throw new RpcException(new Status(StatusCode.AlreadyExists, "User with this email already exists"));
        }
        
        var user = new User
        {
            Email = request.Email.Trim().ToLower(),
            Names = request.Names.Trim(),
        };

        try
        {
            context.Users.Add(user);
            await context.SaveChangesAsync();
            return CreateUserResponse(user);
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to create user"));
        }
    }

    public override async Task<UserResponse> GetUser(GetUserRequest request, ServerCallContext context1)
    {
        var user = await context.Users
            .AsNoTracking()
            .FirstOrDefaultAsync(u => u.Id == request.Id);

        if (user == null || user.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));
        }

        return CreateUserResponse(user);
    }

    public override async Task<UserResponse> UpdateUser(UpdateUserRequest request, ServerCallContext context1)
    {
        if (request.Id <= 0)
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "Invalid user ID"));
        }

        var user = await context.Users
            .FirstOrDefaultAsync(u => u.Id == request.Id);

        if (user == null || user.DeletedAt != null)
        {
            throw new RpcException(new Status(StatusCode.NotFound, "User not found"));
        }
        
        if (!string.IsNullOrWhiteSpace(request.Email))
        {
            if (!IsValidEmail(request.Email))
            {
                throw new RpcException(new Status(StatusCode.InvalidArgument, "Invalid email format"));
            }

            var normalizedEmail = request.Email.Trim().ToLower();
            if (normalizedEmail != user.Email)
            {
                var emailExists = await context.Users
                    .AnyAsync(u => u.Email == normalizedEmail && u.Id != request.Id && u.DeletedAt == null);

                if (emailExists)
                {
                    throw new RpcException(new Status(StatusCode.AlreadyExists, "Email already in use"));
                }

                user.Email = normalizedEmail;
            }
        }
        
        if (!string.IsNullOrWhiteSpace(request.Names))
        {
            user.Names = request.Names.Trim();
        }

        try
        {
            await context.SaveChangesAsync();
            return CreateUserResponse(user);
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to update user"));
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
        
        var hasActiveStreams = await context.Streams
            .AnyAsync(s => s.UserId == user.Id && 
                          s.DeletedAt == null && 
                          (s.Status == EStreamStatus.ONLINE || s.Status == EStreamStatus.SCHEDULED));

        if (hasActiveStreams)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, 
                "Cannot delete user with active streams. Please end or cancel all streams first."));
        }

        try
        {
            user.DeletedAt = DateTime.UtcNow;
            await context.SaveChangesAsync();
            return new Empty();
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to delete user"));
        }
    }

    public override async Task<ListUsersResponse> ListUsers(Empty request, ServerCallContext context1)
    {
        try
        {
            var users = await context.Users
                .AsNoTracking()
                .Where(u => u.DeletedAt == null)
                .OrderByDescending(u => u.CreatedAt)
                .Select(u => CreateUserResponse(u))
                .ToListAsync();

            var response = new ListUsersResponse();
            response.Users.AddRange(users);
            return response;
        }
        catch
        {
            throw new RpcException(new Status(StatusCode.Internal, "Failed to retrieve users"));
        }
    }

    private static UserResponse CreateUserResponse(User user)
    {
        return new UserResponse
        {
            Id = user.Id,
            Email = user.Email,
            Names = user.Names,
        };
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
}
