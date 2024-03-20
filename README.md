# Financial Exchange v1

This is v1 of the full orderbook, matching engine, and API that I built in Go. It is my first complete Go project, and it has since been expanded upon at my company RealBlock Exchange. 

Anyone who is interested in building or learning about exchange infrastructure should start by understanding this project.

### Where to Start Building

Once you understand how the matching engine works, here are some areas I would begin when thinking about how to expand this project:
- Build mock UI to implement frontend for API
- Focus on reducing local latencies: LMAX Disruptor, migrate to FiX API, proper memory management, etc
- Create your own market makers
- And much more, be creative!

