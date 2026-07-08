package version

// Version is the release version shown in /v1/health (and the UI badge).
// Release builds override this via -ldflags "-X keyrafted/internal/version.Version=v0.x.y".
var Version = "0.3.0"
