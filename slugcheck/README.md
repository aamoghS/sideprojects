# slugcheck

`surf` audits page content; this checks the URL layer. Point it at a site, it reads `sitemap.xml`, follows redirects, and prints slug collisions (multiple sitemap entries landing on the same final URL) plus chains that bounce too many times.

Handy after a Next.js migration when old blog paths and new ones both show up in Search Console.

```bash
cd slugcheck
go build -o slugcheck.exe .
./slugcheck.exe https://example.com
./slugcheck.exe -sitemap https://example.com/sitemap-0.xml https://example.com
```

## System design

How a request or a run moves through the code, then the choices behind that shape. In the decision charts, a **solid** box is the route this repo takes and a **dashed** box is the alternative it turns down; the edge label is the tradeoff.

### Sitemap to a final URL

Start at sitemap.xml , follow each URL, and record the final location and the hop count. A collision is two starts with one end. A chain that bounces too many times is its own flag.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  map["sitemap.xml"]
  hop["Follow redirects"]
  final["Final URL and hop count"]
  hit["Two starts, one end"]
  map --> hop --> final --> hit
  class map,hop,final,hit chosen
```

### The URL, before the HTML

surf scores the page. slugcheck does not fetch the body to do that. A redirect bug would otherwise show up as a duplicate title warning, or not show up at all.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  classDef rejected fill:#f5f5f5,stroke:#9e9e9e,stroke-width:1px,stroke-dasharray:5 5,color:#616161
  q{Two sitemap URLs}
  q -->|"same final location"| urls["Flag the collision"]
  q -->|"fetch the body and score it"| body["surf scores the HTML"]
  class urls chosen
  class body rejected
```

Longer write-up: [aamogh.vercel.app/projects/slugcheck](https://aamogh.vercel.app/projects/slugcheck.html)
