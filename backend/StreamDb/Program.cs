using StreamDb.Context;
using StreamDb.Services;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddNpgsql<StreamDbContext>(builder.Configuration.GetConnectionString("DefaultConnection"));
builder.Services.AddGrpc();

var app = builder.Build();

app.MapGrpcService<UserService>();
app.MapGrpcService<StreamService>();
app.MapGrpcService<CommentService>();
app.MapGet("/",
    () =>
        "Communication with gRPC endpoints must be made through a gRPC client. To learn how to create a client, visit: https://go.microsoft.com/fwlink/?linkid=2086909");

app.Run();