using Microsoft.EntityFrameworkCore;
using StreamDb.Context;
using StreamDb.Services;
using UserService = StreamDb.Services.UserService;

var builder = WebApplication.CreateBuilder(args);

builder.WebHost.UseKestrel(serverOptions =>
{
    serverOptions.ListenAnyIP(5000);
});

builder.Services.AddDbContext<StreamDbContext>(options =>
    options.UseNpgsql(builder.Configuration.GetConnectionString("DefaultConnection")));
builder.Services.AddGrpc();
builder.Services.AddGrpcReflection();
var app = builder.Build();

using (var scope = app.Services.CreateScope())
{
    var db = scope.ServiceProvider.GetRequiredService<StreamDbContext>();
    try
    {
        db.Database.Migrate();
        Console.WriteLine("Database migrations applied successfully");
    }
    catch (Exception ex)
    {
        Console.WriteLine($"An error occurred while applying migrations: {ex.Message}");
        throw; // Re-throw if you want the application to fail on migration error
    }
}


app.MapGrpcReflectionService();
app.MapGrpcService<UserService>();
app.MapGrpcService<StreamService>();
app.MapGrpcService<CommentService>();
app.MapGet("/",
    () => "Communication with gRPC endpoints must be made through a gRPC client.");

app.Run();