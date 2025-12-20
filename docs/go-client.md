# Go Client SDK

Complete guide for using the Keyraft Go client library.

## Installation

```bash
go get github.com/keyraft/keyrafted/pkg/client
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "keyrafted/pkg/client"
)

func main() {
    // Create client
    c := client.NewClient(client.Config{
        BaseURL: "http://localhost:7200",
        Token:   "your-token-here",
        Timeout: 30 * time.Second,
    })

    // Store configuration
    entry, err := c.Set("myapp/prod", "DB_HOST", "localhost", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Stored: %s (v%d)\n", entry.Key, entry.Version)

    // Store secret (encrypted)
    _, err = c.SetSecret("myapp/prod", "DB_PASSWORD", "secret123", nil)
    if err != nil {
        log.Fatal(err)
    }

    // Get configuration
    entry, err = c.Get("myapp/prod", "DB_HOST")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Value: %s\n", entry.Value)
}
```

## Client Methods

### Creating a Client

```go
c := client.NewClient(client.Config{
    BaseURL: "http://localhost:7200",
    Token:   "your-token",
    Timeout: 30 * time.Second,  // Optional, default: 30s
})
```

### Set Configuration

```go
entry, err := c.Set(namespace, key, value, metadata)
```

**Example:**
```go
entry, err := c.Set("myapp/prod", "API_TIMEOUT", "30s", map[string]string{
    "unit": "seconds",
})
```

### Set Secret (Encrypted)

```go
entry, err := c.SetSecret(namespace, key, value, metadata)
```

**Example:**
```go
entry, err := c.SetSecret("myapp/prod", "API_KEY", "sk_live_xxxxx", nil)
```

### Get Value

```go
entry, err := c.Get(namespace, key)
```

**Example:**
```go
entry, err := c.Get("myapp/prod", "API_TIMEOUT")
if err != nil {
    log.Fatal(err)
}
fmt.Println(entry.Value)  // "30s"
```

### Get Specific Version

```go
version, err := c.GetVersion(namespace, key, versionNumber)
```

**Example:**
```go
// Get version 2 of a key
v2, err := c.GetVersion("myapp/prod", "API_TIMEOUT", 2)
fmt.Printf("Version 2 was: %s\n", v2.Value)
```

### List Keys

```go
entries, err := c.List(namespace)
```

**Example:**
```go
entries, err := c.List("myapp/prod")
for _, entry := range entries {
    fmt.Printf("%s = %s\n", entry.Key, entry.Value)
}
```

### Delete Key

```go
err := c.Delete(namespace, key)
```

### Watch for Changes

```go
event, err := c.Watch(namespace, timeout)
```

**Example:**
```go
// Watch for changes (blocks until change or timeout)
event, err := c.Watch("myapp/prod", 30*time.Second)
if err != nil {
    log.Fatal(err)
}

if event.Timeout {
    fmt.Println("No changes")
} else {
    fmt.Printf("Change: %s on %s\n", event.Action, event.Key)
}
```

### Health Check

```go
health, err := c.Health()
```

---

## Cached Client (Recommended)

The cached client provides automatic reloading and change notifications.

### Creating a Cached Client

```go
// Create base client
c := client.NewClient(client.Config{
    BaseURL: "http://localhost:7200",
    Token:   "your-token",
})

// Create cached client
cached, err := client.NewCachedClient(client.CacheConfig{
    Client:       c,
    Namespace:    "myapp/prod",
    PollInterval: 10 * time.Second,  // Check for updates every 10s
})
if err != nil {
    log.Fatal(err)
}
defer cached.Close()
```

### Get from Cache

```go
// Fast - no API call
value, ok := cached.Get("DB_HOST")
if ok {
    fmt.Println(value)
}
```

### Get Full Entry from Cache

```go
entry, ok := cached.GetEntry("DB_HOST")
if ok {
    fmt.Printf("Version: %d, Type: %s\n", entry.Version, entry.Type)
}
```

### Get All Cached Values

```go
allValues := cached.GetAll()
for key, value := range allValues {
    fmt.Printf("%s = %s\n", key, value)
}
```

### Register Change Callback

```go
// Called automatically when config changes
cached.OnChange(func(key, value string) {
    fmt.Printf("Config changed: %s = %s\n", key, value)
    // Reload your application configuration here
})
```

### Update Values

```go
// Update config (cache refreshes automatically)
err := cached.Set("NEW_KEY", "new-value", nil)

// Update secret
err := cached.SetSecret("SECRET_KEY", "secret-value", nil)

// Delete key
err := cached.Delete("OLD_KEY")
```

---

## Complete Example: Web Application

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "time"
    
    "keyrafted/pkg/client"
)

type Config struct {
    mu         sync.RWMutex
    apiURL     string
    apiKey     string
    timeout    string
}

func (c *Config) Update(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    switch key {
    case "API_URL":
        c.apiURL = value
        log.Printf("Updated API_URL: %s", value)
    case "API_KEY":
        c.apiKey = value
        log.Println("Updated API_KEY")
    case "TIMEOUT":
        c.timeout = value
        log.Printf("Updated TIMEOUT: %s", value)
    }
}

func (c *Config) GetAPIURL() string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.apiURL
}

func main() {
    // Create Keyraft client
    baseClient := client.NewClient(client.Config{
        BaseURL: "http://localhost:7200",
        Token:   os.Getenv("KEYRAFT_TOKEN"),
        Timeout: 30 * time.Second,
    })

    // Create cached client
    cached, err := client.NewCachedClient(client.CacheConfig{
        Client:       baseClient,
        Namespace:    "myapp/prod",
        PollInterval: 10 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer cached.Close()

    // Application config
    config := &Config{}

    // Load initial config
    if val, ok := cached.Get("API_URL"); ok {
        config.apiURL = val
    }
    if val, ok := cached.Get("API_KEY"); ok {
        config.apiKey = val
    }
    if val, ok := cached.Get("TIMEOUT"); ok {
        config.timeout = val
    }

    // Register change handler
    cached.OnChange(func(key, value string) {
        config.Update(key, value)
    })

    // HTTP handler
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "API URL: %s\n", config.GetAPIURL())
        fmt.Fprintf(w, "Config loaded from Keyraft with auto-reload!\n")
    })

    // Start server
    srv := &http.Server{Addr: ":8080"}
    go func() {
        log.Println("Server starting on :8080")
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    // Wait for interrupt
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt)
    <-stop

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

---

## Best Practices

### 1. Use Cached Client for Applications

```go
// ✅ Good - auto-reload, fast reads
cached, _ := client.NewCachedClient(config)
defer cached.Close()

// ❌ Avoid - manual API calls every time
c := client.NewClient(config)
value, _ := c.Get("ns", "key")
```

### 2. Handle Configuration Changes

```go
cached.OnChange(func(key, value string) {
    // Reload application state
    // Update connection pools
    // Restart services if needed
})
```

### 3. Use Metadata for Context

```go
c.Set("myapp/prod", "FEATURE_FLAG", "true", map[string]string{
    "enabled_by": "john@example.com",
    "enabled_at": time.Now().Format(time.RFC3339),
    "reason": "testing new feature",
})
```

### 4. Store Secrets Safely

```go
// ✅ Secrets are encrypted
c.SetSecret("myapp/prod", "DB_PASSWORD", password, nil)

// ❌ Don't store secrets as config
c.Set("myapp/prod", "DB_PASSWORD", password, nil)
```

### 5. Handle Errors Gracefully

```go
entry, err := c.Get("myapp/prod", "DB_HOST")
if err != nil {
    // Fallback to default
    dbHost = "localhost"
    log.Printf("Using default DB_HOST: %v", err)
} else {
    dbHost = entry.Value
}
```

### 6. Close Cached Clients

```go
cached, _ := client.NewCachedClient(config)
defer cached.Close()  // Important: stops background goroutines
```

---

## Error Handling

```go
entry, err := c.Get("myapp/prod", "KEY")
if err != nil {
    // Check error message
    if strings.Contains(err.Error(), "404") {
        // Key not found
    } else if strings.Contains(err.Error(), "401") {
        // Authentication failed
    } else if strings.Contains(err.Error(), "403") {
        // Insufficient permissions
    } else {
        // Other error
    }
}
```

---

## Thread Safety

Both `Client` and `CachedClient` are thread-safe and can be used concurrently:

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(n int) {
        defer wg.Done()
        key := fmt.Sprintf("KEY_%d", n)
        cached.Get(key)  // Safe concurrent access
    }(i)
}

wg.Wait()
```

