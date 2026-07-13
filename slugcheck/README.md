# slugcheck

`surf` audits page content; this checks the URL layer. Point it at a site, it reads `sitemap.xml`, follows redirects, and prints slug collisions (multiple sitemap entries landing on the same final URL) plus chains that bounce too many times.

Handy after a Next.js migration when old blog paths and new ones both show up in Search Console.

```bash
cd slugcheck
go build -o slugcheck.exe .
./slugcheck.exe https://example.com
./slugcheck.exe -sitemap https://example.com/sitemap-0.xml https://example.com
```
