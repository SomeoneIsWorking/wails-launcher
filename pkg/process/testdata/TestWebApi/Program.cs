var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

app.MapGet("/health", () => Results.Ok(new { status = "healthy" }));

app.MapGet("/throw", (HttpContext ctx) =>
{
    throw new InvalidOperationException("Intentional test exception from /throw endpoint");
});

app.Run();
