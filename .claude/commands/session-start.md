Initialize vibe coding session with context and sector awareness:

1. CONTEXT LOADING SEQUENCE:
   ```
   Loading priority:
   1. CLAUDE.md (project rules)
   2. ACTIVE_CONTEXT.md (previous state if exists)
   3. GIT_CONTEXT.md (current branch, last commits)
   4. SECTOR_PLAN.md (active boundaries if exists)
   ```

2. ESTABLISH SESSION PARAMETERS:
   ```markdown
   SESSION INITIALIZED: [timestamp]
   Branch: [current git branch]
   Last Commit: [hash and message]
   Active Sector: [if any]
   Token Budget: Fresh (0%)
   Context Strategy: Aggressive preservation until 70%
   ```

3. PROMPT FOR FOCUS:
   "What's our focus? (Will determine primary sector)"

4. SECTOR ANALYSIS:
   Based on response, identify:
   - Primary sector for work
   - Allowed auxiliary sectors
   - Forbidden sectors

5. LOG SESSION START:
   Update SESSION_LOG.md with initialization

Response: "Session ready. Context loaded. Sector identified: [sector]. Ready for vibe coding."