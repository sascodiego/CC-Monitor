Smart checkpoint with dual commit capability:

1. ANALYZE WHAT NEEDS SAVING:
   ```bash
   # Check for changes in both sector and context
   SECTOR_CHANGES=$(git diff --name-only | grep -v "^\.claude/" | wc -l)
   CONTEXT_CHANGES=$(git diff --name-only | grep "^\.claude/context/" | wc -l)
   
   echo "📊 Changes detected:"
   echo "  Sector files: $SECTOR_CHANGES"
   echo "  Context files: $CONTEXT_CHANGES"
   ```

2. DUAL COMMIT STRATEGY:
   ```bash
   # If both sector and context have changes
   if [[ $SECTOR_CHANGES -gt 0 && $CONTEXT_CHANGES -gt 0 ]]; then
     echo "📦 Creating dual checkpoint (sector + context)"
     
     # First: Commit sector changes
     git add $(git diff --name-only | grep -v "^\.claude/")
     git commit -m "WIP: [$ACTIVE_SECTOR] - checkpoint"
     
     # Second: Commit context changes
     git add .claude/context/*.md
     git commit -m "context: checkpoint for $ACTIVE_SECTOR work"
     
   # If only sector changes
   elif [[ $SECTOR_CHANGES -gt 0 ]]; then
     git add $(git diff --name-only | grep -v "^\.claude/")
     git commit -m "WIP: [$ACTIVE_SECTOR] - checkpoint"
     
   # If only context changes
   elif [[ $CONTEXT_CHANGES -gt 0 ]]; then
     git add .claude/context/*.md
     git commit -m "context: checkpoint"
   fi
   ```

3. CONTEXT HEALTH CHECK:
   ```
   Token Usage: ~[estimate]%
   Working Memory: [files in context]
   Active Sector: [current]
   ```

4. SECTOR BOUNDARY VALIDATION:
   ```bash
   # Verify no violations in sector commits
   git diff --name-only | while read file; do
     if [[ ! "$file" =~ ^${ACTIVE_SECTOR}/ && ! "$file" =~ ^\.claude/ ]]; then
       echo "VIOLATION: $file outside sector"
     fi
   done
   ```

5. UPDATE TRACKING:
   ```markdown
   [HH:MM] CHECKPOINT
   - Sector changes: $SECTOR_CHANGES files
   - Context changes: $CONTEXT_CHANGES files
   - Tokens: ~[estimate]%
   - Next: [planned action]
   ```

6. RESPONSE:
   "✓ Checkpoint saved (sector: $SECTOR_CHANGES files, context: $CONTEXT_CHANGES files)"

7. Reload to your context rules in CLAUDE.md