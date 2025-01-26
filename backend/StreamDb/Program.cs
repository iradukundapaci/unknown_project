using Microsoft.EntityFrameworkCore;
using StreamDb.Context;
using StreamDb.Services;

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

app.MapGrpcReflectionService();
app.MapGrpcService<UserService>();
app.MapGrpcService<StreamService>();
app.MapGrpcService<CommentService>();
app.MapGet("/",
    () => "Communication with gRPC endpoints must be made through a gRPC client.");

app.Run();