package vlr

// CacheVersion is the parser/schema generation stamped into the cache. Bump it
// whenever a parser or serialized-shape change makes previously-cached JSON
// stale; on startup the fetcher wipes any cache written by an older generation
// so stale entries are regenerated from the improved parser instead of lingering
// until someone clears the DB by hand.
const CacheVersion = 1
