// Command csharp-rest-api generates a minimal, working C# HTTP API
// service. The plugin itself is Go (there's no C# SDK yet — see
// docs/architecture/roadmap.md), but what it generates is C# source.
// Same quality bar as go-rest-api/node-rest-api (ADR-0004): the
// generated project must actually build and pass its own tests,
// standalone, offline.
//
// Uses System.Net.HttpListener (BCL, no NuGet packages) rather than
// ASP.NET Core: the plain "Microsoft.NET.Sdk" console-app SDK needs
// nothing beyond the .NET SDK itself, unlike ASP.NET Core's heavier
// project templates and their own package references. There's no
// bundled test framework either (avoiding a NuGet/xUnit fetch) — the
// same program takes a "test" argument and runs hand-rolled checks
// instead of starting the server, matching the pattern the other
// hand-rolled-test templates (Rust/Java/Kotlin/C++) use.
//
// NOT LOCALLY BUILDABLE/TESTABLE IN THIS ENVIRONMENT: no working .NET
// SDK was available where this template was written (only a broken
// `dotnet` on PATH). Written carefully against standard
// HttpListener/HttpClient idioms and cross-checked against the .NET
// documentation, but the first real verification of the generated
// project is this repository's CI run
// (TestEndToEndGenerateCSharpRestAPI), not a local one. Treat an
// initial red CI run as expected, not a process failure.
package main

import (
	"fmt"

	sdk "github.com/intruder0007/Lumo/sdk/go/sdk"
	"github.com/intruder0007/Lumo/sdk/go/sdk/fsutil"
)

type csharpRestAPITemplate struct{}

func (csharpRestAPITemplate) Generate(req sdk.GenerateRequest) (sdk.GenerateResponse, error) {
	files := map[string]string{
		"Server.csproj": serverCsprojFile(req.ProjectName),
		"Program.cs":    programCSFile(),
		"README.md":     readmeFile(req.ProjectName),
		".gitignore":    gitignoreFile(),
	}

	written, err := fsutil.WriteFiles(req.TargetDir, files)
	if err != nil {
		return sdk.GenerateResponse{}, err
	}

	return sdk.GenerateResponse{
		FilesWritten: written,
		NextSteps: []string{
			fmt.Sprintf("cd %s", req.ProjectName),
			"dotnet run -- test # runs the tests",
			"dotnet run # serves on :8080, GET /healthz",
		},
	}, nil
}

func serverCsprojFile(projectName string) string {
	return fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">

  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
    <RootNamespace>%s</RootNamespace>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <InvariantGlobalization>true</InvariantGlobalization>
  </PropertyGroup>

</Project>
`, sanitizeNamespace(projectName))
}

// sanitizeNamespace turns a project name into a valid-enough C#
// identifier for RootNamespace: hyphens (the one non-identifier
// character project names are allowed to contain, see
// core/config.Answers.Validate) become underscores.
func sanitizeNamespace(projectName string) string {
	out := make([]rune, 0, len(projectName))
	for _, r := range projectName {
		if r == '-' {
			out = append(out, '_')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func programCSFile() string {
	return `using System.Net;
using System.Net.Http;
using System.Net.Sockets;
using System.Text;

if (args.Length > 0 && args[0] == "test")
{
    return await RunTestsAsync();
}

var server = CreateListener(8080);
server.Start();
Console.WriteLine("listening on :8080");
await ServeAsync(server);
return 0;

static HttpListener CreateListener(int port)
{
    var listener = new HttpListener();
    listener.Prefixes.Add($"http://localhost:{port}/");
    return listener;
}

static async Task ServeAsync(HttpListener listener)
{
    while (true)
    {
        var context = await listener.GetContextAsync();
        await HandleRequestAsync(context);
    }
}

static async Task HandleRequestAsync(HttpListenerContext context)
{
    var request = context.Request;
    var response = context.Response;

    if (request.HttpMethod == "GET" && request.Url?.AbsolutePath == "/healthz")
    {
        byte[] body = Encoding.UTF8.GetBytes("{\"status\":\"ok\"}");
        response.ContentType = "application/json";
        response.ContentLength64 = body.Length;
        await response.OutputStream.WriteAsync(body);
        response.Close();
        return;
    }

    response.StatusCode = 404;
    response.Close();
}

// GetFreePort asks the OS for an ephemeral port via a throwaway
// TcpListener, then releases it immediately — HttpListener itself has
// no "bind to port 0" option the way raw sockets do, so this is the
// standard way to get a free port for a test run.
static int GetFreePort()
{
    var probe = new TcpListener(IPAddress.Loopback, 0);
    probe.Start();
    int port = ((IPEndPoint)probe.LocalEndpoint).Port;
    probe.Stop();
    return port;
}

static async Task<int> RunTestsAsync()
{
    int port = GetFreePort();
    var server = CreateListener(port);
    server.Start();
    _ = ServeAsync(server);

    try
    {
        using var client = new HttpClient();
        var response = await client.GetAsync($"http://localhost:{port}/healthz");
        var body = await response.Content.ReadAsStringAsync();

        Check(response.StatusCode == HttpStatusCode.OK, $"expected status 200, got {(int)response.StatusCode}");
        Check(body == "{\"status\":\"ok\"}", $"expected body {{\"status\":\"ok\"}}, got {body}");
    }
    finally
    {
        server.Stop();
    }

    Console.WriteLine("all tests passed");
    return 0;
}

static void Check(bool condition, string message)
{
    if (!condition)
    {
        throw new Exception(message);
    }
}
`
}

func readmeFile(projectName string) string {
	return fmt.Sprintf(`# %s

A C# HTTP API service, generated by Lumo.

## Run

`+"```sh\ndotnet run\n```"+`

Serves on :8080. `+"`GET /healthz`"+` returns `+"`{\"status\":\"ok\"}`"+`.

## Test

`+"```sh\ndotnet run -- test\n```"+`

Uses only `+"`System.Net.HttpListener`"+`/`+"`System.Net.Http.HttpClient`"+` (BCL, no
NuGet packages) and the plain console-app SDK (not ASP.NET Core) — no
package restore beyond the .NET SDK itself. `+"`dotnet run -- test`"+` runs
hand-rolled checks instead of starting the server (no xUnit/NUnit
fetch needed either).
`, projectName)
}

func gitignoreFile() string {
	return "/bin/\n/obj/\n"
}

func main() {
	sdk.Serve(csharpRestAPITemplate{})
}
