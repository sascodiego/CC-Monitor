/**
 * CONTEXT:   Smart query cache system for reporting performance optimization
 * INPUT:     Query results, cache keys, TTL values, and cache invalidation triggers
 * OUTPUT:    High-performance caching with automatic expiration and memory management
 * BUSINESS:  Fast report generation requires caching to achieve < 100ms response times
 * CHANGE:    Initial implementation with LRU cache and smart TTL management
 * RISK:      Medium - Cache invalidation logic must be correct to prevent stale data
 */

package database

import (
	"container/list"
	"sync"
	"time"
)

/**
 * CONTEXT:   Cache entry with value, expiration, and access tracking
 * INPUT:     Cached data, expiration time, and access metadata
 * OUTPUT:    Cache entry suitable for LRU eviction and expiration
 * BUSINESS:  Cache entries must track both temporal and usage-based expiration
 * CHANGE:    Initial cache entry structure with comprehensive tracking
 * RISK:      Low - Simple data structure for cache management
 */
type CacheEntry struct {
	Key        string
	Value      interface{}
	ExpiresAt  time.Time
	AccessedAt time.Time
	HitCount   int64
	Size       int64 // Approximate memory usage in bytes
}

/**
 * CONTEXT:   LRU cache with TTL and memory-based eviction for query results
 * INPUT:     Cache operations, memory limits, and expiration policies
 * OUTPUT:    High-performance cache with automatic cleanup and optimization
 * BUSINESS:  Smart caching enables sub-100ms query response times for reports
 * CHANGE:    Initial implementation with LRU + TTL hybrid approach
 * RISK:      Medium - Complex cache logic requires careful synchronization
 */
type QueryCache struct {
	maxSize      int64                    // Maximum memory usage in bytes
	currentSize  int64                    // Current memory usage
	entries      map[string]*CacheEntry   // Key -> CacheEntry mapping
	accessOrder  *list.List               // LRU order tracking
	elementMap   map[string]*list.Element // Key -> List element mapping
	hitCount     int64                    // Total cache hits
	missCount    int64                    // Total cache misses
	evictCount   int64                    // Total evictions
	mu           sync.RWMutex             // Concurrent access protection
}

// Cache configuration
const (
	DefaultMaxCacheSize     = 100 * 1024 * 1024 // 100MB
	DefaultCleanupInterval  = 5 * time.Minute
	MaxCacheEntries         = 1000
	ReportSizeEstimate      = 50 * 1024  // 50KB per report estimate
)

/**
 * CONTEXT:   Create new query cache with memory and performance optimization
 * INPUT:     Cache size limits and cleanup configuration
 * OUTPUT:    Initialized cache ready for high-performance query caching
 * BUSINESS:  Query cache initialization with reasonable defaults for work tracking
 * CHANGE:    Initial cache constructor with background cleanup
 * RISK:      Low - Standard cache initialization with safe defaults
 */
func NewQueryCache() *QueryCache {
	cache := &QueryCache{
		maxSize:     DefaultMaxCacheSize,
		currentSize: 0,
		entries:     make(map[string]*CacheEntry),
		accessOrder: list.New(),
		elementMap:  make(map[string]*list.Element),
		hitCount:    0,
		missCount:   0,
		evictCount:  0,
	}

	// Start background cleanup goroutine
	go cache.backgroundCleanup()

	return cache
}

/**
 * CONTEXT:   Store query result in cache with TTL and memory management
 * INPUT:     Cache key, query result value, and time-to-live duration
 * OUTPUT:    Cached entry with automatic expiration and LRU tracking
 * BUSINESS:  Fast cache storage for frequently accessed reports
 * CHANGE:    Initial implementation with memory-aware caching
 * RISK:      Medium - Memory estimation and eviction logic must be accurate
 */
func (qc *QueryCache) Set(key string, value interface{}, ttl time.Duration) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	now := time.Now()
	estimatedSize := qc.estimateSize(value)

	// Check if key already exists
	if existingEntry, exists := qc.entries[key]; exists {
		// Update existing entry
		qc.currentSize -= existingEntry.Size
		existingEntry.Value = value
		existingEntry.ExpiresAt = now.Add(ttl)
		existingEntry.AccessedAt = now
		existingEntry.Size = estimatedSize
		qc.currentSize += estimatedSize

		// Move to front of LRU list
		if element := qc.elementMap[key]; element != nil {
			qc.accessOrder.MoveToFront(element)
		}
		return
	}

	// Create new entry
	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		ExpiresAt:  now.Add(ttl),
		AccessedAt: now,
		HitCount:   0,
		Size:       estimatedSize,
	}

	// Add to maps and LRU list
	qc.entries[key] = entry
	element := qc.accessOrder.PushFront(entry)
	qc.elementMap[key] = element
	qc.currentSize += estimatedSize

	// Ensure cache size limits
	qc.evictIfNecessary()
}

/**
 * CONTEXT:   Retrieve cached query result with LRU update
 * INPUT:     Cache key for lookup
 * OUTPUT:    Cached value if found and not expired, nil otherwise
 * BUSINESS:  Fast cache retrieval for report generation performance
 * CHANGE:    Initial implementation with access tracking and expiration
 * RISK:      Low - Standard cache get operation with expiration check
 */
func (qc *QueryCache) Get(key string) interface{} {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	entry, exists := qc.entries[key]
	if !exists {
		qc.missCount++
		return nil
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, remove it
		qc.deleteEntry(key)
		qc.missCount++
		return nil
	}

	// Update access tracking
	entry.AccessedAt = time.Now()
	entry.HitCount++
	qc.hitCount++

	// Move to front of LRU list
	if element := qc.elementMap[key]; element != nil {
		qc.accessOrder.MoveToFront(element)
	}

	return entry.Value
}

/**
 * CONTEXT:   Remove specific entry from cache
 * INPUT:     Cache key to remove
 * OUTPUT:    Boolean indicating if entry was found and removed
 * BUSINESS:  Manual cache invalidation for data consistency
 * CHANGE:    Initial implementation with complete cleanup
 * RISK:      Low - Standard cache deletion with resource cleanup
 */
func (qc *QueryCache) Delete(key string) bool {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	return qc.deleteEntry(key)
}

/**
 * CONTEXT:   Internal entry deletion with resource cleanup
 * INPUT:     Cache key for entry to delete
 * OUTPUT:    Boolean indicating successful deletion
 * BUSINESS:  Clean entry removal prevents memory leaks
 * CHANGE:    Initial implementation with comprehensive cleanup
 * RISK:      Low - Internal deletion with proper resource management
 */
func (qc *QueryCache) deleteEntry(key string) bool {
	entry, exists := qc.entries[key]
	if !exists {
		return false
	}

	// Remove from maps
	delete(qc.entries, key)
	qc.currentSize -= entry.Size

	// Remove from LRU list
	if element := qc.elementMap[key]; element != nil {
		qc.accessOrder.Remove(element)
		delete(qc.elementMap, key)
	}

	return true
}

/**
 * CONTEXT:   Invalidate cache entries matching pattern or criteria
 * INPUT:     Invalidation pattern (user ID, date prefix, etc.)
 * OUTPUT:    Number of entries invalidated
 * BUSINESS:  Smart invalidation prevents stale data in reports
 * CHANGE:    Initial implementation with pattern-based invalidation
 * RISK:      Medium - Pattern matching logic must be efficient and correct
 */
func (qc *QueryCache) InvalidatePattern(pattern string) int {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	invalidatedCount := 0
	keysToDelete := make([]string, 0)

	// Find matching keys
	for key := range qc.entries {
		if qc.matchesPattern(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	// Delete matching entries
	for _, key := range keysToDelete {
		if qc.deleteEntry(key) {
			invalidatedCount++
		}
	}

	return invalidatedCount
}

/**
 * CONTEXT:   Check if cache key matches invalidation pattern
 * INPUT:     Cache key and invalidation pattern
 * OUTPUT:    Boolean indicating if key should be invalidated
 * BUSINESS:  Pattern matching enables targeted cache invalidation
 * CHANGE:    Initial implementation with common pattern types
 * RISK:      Low - Simple pattern matching for cache invalidation
 */
func (qc *QueryCache) matchesPattern(key, pattern string) bool {
	// Simple pattern matching for now
	// Could be extended with regex or more sophisticated matching
	
	// Pattern examples:
	// "user_12345" - invalidate all entries for user
	// "daily_report" - invalidate all daily reports
	// "2024-01" - invalidate all entries for January 2024
	
	return key == pattern || 
		   (len(pattern) > 0 && len(key) > len(pattern) && key[:len(pattern)] == pattern)
}

/**
 * CONTEXT:   LRU eviction when cache size exceeds limits
 * INPUT:     No parameters, uses current cache state
 * OUTPUT:     Evicted entries to maintain size and count limits
 * BUSINESS:  Memory management ensures cache doesn't consume excessive resources
 * CHANGE:    Initial implementation with size and count-based eviction
 * RISK:      Medium - Eviction logic must balance performance and memory usage
 */
func (qc *QueryCache) evictIfNecessary() {
	// Evict expired entries first
	qc.evictExpired()

	// Evict LRU entries if over size limit
	for qc.currentSize > qc.maxSize || len(qc.entries) > MaxCacheEntries {
		if qc.accessOrder.Len() == 0 {
			break
		}

		// Get least recently used entry
		lruElement := qc.accessOrder.Back()
		if lruElement == nil {
			break
		}

		lruEntry := lruElement.Value.(*CacheEntry)
		if qc.deleteEntry(lruEntry.Key) {
			qc.evictCount++
		}
	}
}

/**
 * CONTEXT:   Remove all expired entries from cache
 * INPUT:     No parameters, checks all entries against current time
 * OUTPUT:    Number of expired entries removed
 * BUSINESS:  Expired entry cleanup maintains cache accuracy and frees memory
 * CHANGE:    Initial implementation with timestamp-based expiration
 * RISK:      Low - Standard expiration cleanup
 */
func (qc *QueryCache) evictExpired() int {
	now := time.Now()
	expiredCount := 0
	keysToDelete := make([]string, 0)

	// Find expired entries
	for key, entry := range qc.entries {
		if now.After(entry.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	// Delete expired entries
	for _, key := range keysToDelete {
		if qc.deleteEntry(key) {
			expiredCount++
		}
	}

	return expiredCount
}

/**
 * CONTEXT:   Background goroutine for periodic cache cleanup
 * INPUT:     No parameters, runs continuously with cleanup interval
 * OUTPUT:    Automatic cleanup of expired entries and memory management
 * BUSINESS:  Background cleanup maintains cache health without blocking queries
 * CHANGE:    Initial implementation with configurable cleanup interval
 * RISK:      Low - Background cleanup with minimal performance impact
 */
func (qc *QueryCache) backgroundCleanup() {
	ticker := time.NewTicker(DefaultCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			qc.mu.Lock()
			expiredCount := qc.evictExpired()
			qc.mu.Unlock()
			
			if expiredCount > 0 {
				// Log cleanup activity for monitoring
				// log.Printf("Cache cleanup: removed %d expired entries", expiredCount)
			}
		}
	}
}

/**
 * CONTEXT:   Estimate memory usage of cached values
 * INPUT:     Value to be cached (report structures, query results)
 * OUTPUT:    Estimated memory size in bytes
 * BUSINESS:  Memory estimation enables size-based cache eviction
 * CHANGE:    Initial implementation with type-specific size estimates
 * RISK:      Medium - Size estimation accuracy affects cache performance
 */
func (qc *QueryCache) estimateSize(value interface{}) int64 {
	switch v := value.(type) {
	case *DailyReport:
		// Base size + project breakdown + hourly breakdown
		size := int64(200) // Base struct size
		size += int64(len(v.ProjectBreakdown)) * 100 // Projects
		size += int64(len(v.HourlyBreakdown)) * 50   // Hours
		return size
		
	case *WeeklyReport:
		// Base size + daily reports + project breakdown
		size := int64(300) // Base struct size
		size += int64(len(v.DailyBreakdown)) * 200  // Daily reports
		size += int64(len(v.TopProjects)) * 100     // Projects
		return size
		
	case *MonthlyReport:
		// Base size + weekly reports + calendar data + projects
		size := int64(500) // Base struct size
		size += int64(len(v.WeeklyBreakdown)) * 300  // Weekly reports
		size += int64(len(v.CalendarHeatmap)) * 80   // Calendar days
		size += int64(len(v.TopProjects)) * 100      // Projects
		return size
		
	case *ProjectReport:
		// Base size + daily breakdown + hourly pattern
		size := int64(250) // Base struct size
		size += int64(len(v.DailyBreakdown)) * 80   // Daily stats
		size += int64(len(v.HourlyPattern)) * 50    // Hourly stats
		return size
		
	case []ProjectTime:
		// Array of project time data
		return int64(len(v)) * 100
		
	case []HourlyStats:
		// Array of hourly statistics
		return int64(len(v)) * 50
		
	default:
		// Default estimate for unknown types
		return ReportSizeEstimate
	}
}

/**
 * CONTEXT:   Get comprehensive cache statistics for monitoring
 * INPUT:     No parameters, returns current cache state
 * OUTPUT:    Cache performance metrics and memory usage stats
 * BUSINESS:  Cache monitoring enables performance optimization and troubleshooting
 * CHANGE:    Initial implementation with comprehensive metrics
 * RISK:      Low - Read-only statistics collection
 */
type CacheStats struct {
	EntryCount      int                `json:"entry_count"`
	CurrentSize     int64              `json:"current_size_bytes"`
	MaxSize         int64              `json:"max_size_bytes"`
	MemoryUsage     float64            `json:"memory_usage_percent"`
	HitCount        int64              `json:"hit_count"`
	MissCount       int64              `json:"miss_count"`
	EvictCount      int64              `json:"evict_count"`
	HitRate         float64            `json:"hit_rate_percent"`
	TopKeys         []string           `json:"top_keys"`
	ExpirationTimes map[string]time.Time `json:"expiration_times"`
}

func (qc *QueryCache) GetStats() CacheStats {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	totalRequests := qc.hitCount + qc.missCount
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(qc.hitCount) / float64(totalRequests) * 100.0
	}

	memoryUsage := 0.0
	if qc.maxSize > 0 {
		memoryUsage = float64(qc.currentSize) / float64(qc.maxSize) * 100.0
	}

	// Get top accessed keys
	topKeys := qc.getTopKeys(10)

	// Get expiration times for monitoring
	expirations := make(map[string]time.Time)
	for key, entry := range qc.entries {
		expirations[key] = entry.ExpiresAt
	}

	return CacheStats{
		EntryCount:      len(qc.entries),
		CurrentSize:     qc.currentSize,
		MaxSize:         qc.maxSize,
		MemoryUsage:     memoryUsage,
		HitCount:        qc.hitCount,
		MissCount:       qc.missCount,
		EvictCount:      qc.evictCount,
		HitRate:         hitRate,
		TopKeys:         topKeys,
		ExpirationTimes: expirations,
	}
}

/**
 * CONTEXT:   Get most frequently accessed cache keys
 * INPUT:     Number of top keys to return
 * OUTPUT:    Array of cache keys sorted by access frequency
 * BUSINESS:  Top keys analysis helps optimize cache policies
 * CHANGE:    Initial implementation with hit count sorting
 * RISK:      Low - Statistics collection for cache optimization
 */
func (qc *QueryCache) getTopKeys(limit int) []string {
	type KeyHits struct {
		Key   string
		Hits  int64
	}

	keyHits := make([]KeyHits, 0, len(qc.entries))
	for key, entry := range qc.entries {
		keyHits = append(keyHits, KeyHits{
			Key:  key,
			Hits: entry.HitCount,
		})
	}

	// Simple sorting by hit count
	for i := 0; i < len(keyHits); i++ {
		for j := i + 1; j < len(keyHits); j++ {
			if keyHits[i].Hits < keyHits[j].Hits {
				keyHits[i], keyHits[j] = keyHits[j], keyHits[i]
			}
		}
	}

	// Return top keys up to limit
	topKeys := make([]string, 0, limit)
	for i := 0; i < len(keyHits) && i < limit; i++ {
		topKeys = append(topKeys, keyHits[i].Key)
	}

	return topKeys
}

/**
 * CONTEXT:   Clear all entries from cache
 * INPUT:     No parameters, removes all cached data
 * OUTPUT:    Number of entries cleared
 * BUSINESS:  Cache clearing for maintenance or testing purposes
 * CHANGE:    Initial implementation with complete cleanup
 * RISK:      Low - Complete cache reset operation
 */
func (qc *QueryCache) Clear() int {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	entryCount := len(qc.entries)

	// Clear all data structures
	qc.entries = make(map[string]*CacheEntry)
	qc.accessOrder = list.New()
	qc.elementMap = make(map[string]*list.Element)
	qc.currentSize = 0
	qc.hitCount = 0
	qc.missCount = 0
	qc.evictCount = 0

	return entryCount
}

/**
 * CONTEXT:   Preload cache with frequently accessed reports
 * INPUT:     User ID and date range for preloading common reports
 * OUTPUT:    Number of entries preloaded into cache
 * BUSINESS:  Cache preloading improves perceived performance for common queries
 * CHANGE:    Initial implementation with common report patterns
 * RISK:      Medium - Preloading logic must not overwhelm cache or database
 */
func (qc *QueryCache) PreloadCommonReports(userID string, rq *ReportingQueries) int {
	// This would preload common reports like:
	// - Today's daily report
	// - Current week's weekly report
	// - Current month's monthly report
	// - Top projects for current period
	
	// Implementation would depend on the ReportingQueries interface
	// For now, return 0 as placeholder
	return 0
}

/**
 * CONTEXT:   Smart TTL calculation based on report type and recency
 * INPUT:     Cache key and report type for TTL optimization
 * OUTPUT:    Optimized TTL duration for different report types
 * BUSINESS:  Smart TTL reduces cache misses while ensuring data freshness
 * CHANGE:    Initial implementation with report-type-specific TTLs
 * RISK:      Low - TTL optimization based on report characteristics
 */
func (qc *QueryCache) CalculateSmartTTL(key string, reportType string) time.Duration {
	// Current day reports: 5 minutes (data changes frequently)
	if reportType == "daily" && qc.isCurrentDay(key) {
		return 5 * time.Minute
	}
	
	// Current week reports: 15 minutes
	if reportType == "weekly" && qc.isCurrentWeek(key) {
		return 15 * time.Minute
	}
	
	// Current month reports: 30 minutes
	if reportType == "monthly" && qc.isCurrentMonth(key) {
		return 30 * time.Minute
	}
	
	// Historical reports: 4 hours (data doesn't change)
	return 4 * time.Hour
}

// Helper functions for smart TTL calculation
func (qc *QueryCache) isCurrentDay(key string) bool {
	// Parse key to determine if it's for today
	// This would be implemented based on key format
	return false // Placeholder
}

func (qc *QueryCache) isCurrentWeek(key string) bool {
	// Parse key to determine if it's for current week
	return false // Placeholder
}

func (qc *QueryCache) isCurrentMonth(key string) bool {
	// Parse key to determine if it's for current month
	return false // Placeholder
}