---
title: "Mermaid"
description: "Diagrams and charts from text definitions."
sidebar:
  order: 20
---

Mermaid renders diagrams from text definitions inside fenced code blocks tagged with `mermaid`. Flowcharts, sequence diagrams, class diagrams, Gantt charts, and more are supported. Mermaid.js is loaded on-demand — only pages containing a mermaid block load the library.

## Flowchart

````md
```mermaid
flowchart TD
    A[Write Markdown] --> B{Run sarde dev}
    B --> C[Site built]
    C --> D[Open http://localhost:4727]
```
````

```mermaid
flowchart TD
    A[Write Markdown] --> B{Run sarde dev}
    B --> C[Site built]
    C --> D[Open http://localhost:4727]
```

## Sequence Diagram

````md
```mermaid
sequenceDiagram
    Browser->>Dev Server: GET /docs/
    Dev Server->>Engine: Build page
    Engine-->>Dev Server: HTML
    Dev Server-->>Browser: 200 OK
    Note over Browser: Page renders
```
````

```mermaid
sequenceDiagram
    Browser->>Dev Server: GET /docs/
    Dev Server->>Engine: Build page
    Engine-->>Dev Server: HTML
    Dev Server-->>Browser: 200 OK
    Note over Browser: Page renders
```

## Class Diagram

````md
```mermaid
classDiagram
    class SiteBuilder {
        +Build(ctx) error
    }
    class ContentDiscoverer {
        +Discover(root) []Page
    }
    SiteBuilder --> ContentDiscoverer
```
````

```mermaid
classDiagram
    class SiteBuilder {
        +Build(ctx) error
    }
    class ContentDiscoverer {
        +Discover(root) []Page
    }
    SiteBuilder --> ContentDiscoverer
```

For the full syntax reference and all supported diagram types, see the [Mermaid documentation](https://mermaid.js.org/intro/).
