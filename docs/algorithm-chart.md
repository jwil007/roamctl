```mermaid
flowchart TD
    A([Connected to AP]) --> B[Poll signal quality at interval (default 250ms)]
    B --> C{Signal quality?}

    C -->|Excellent| D[No scan · No roam]
    D 

    C -->|Fair| E[Periodic background scan]
    E --> H

    C -->|Degraded| F[Targeted scan on known channels\nfull sweep if environment changed]
    F --> H

    C -->|Critical| G[Immediate full sweep\nacross all channels]
    G --> H

    H[Score all visible APs\nRSSI · SNR · band · width · load · gen] --> I{Better AP above\nscore threshold?}

    I -->|No| J[Stay put · await next scan]
    J 

    I -->|Yes| K[Roam to highest-scored AP]
    K 
```
