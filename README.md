# Go Short TID

Generate short string IDs with a time and random component.

Useful for when you want to *guarantee* 0 collisions between two points in time, while minimizing collisions for generated IDs *within* that time.

**For example:** "Generate base-62 IDs with a time granularity of 1 millisecond, and with 5 extra random characters at the end."


