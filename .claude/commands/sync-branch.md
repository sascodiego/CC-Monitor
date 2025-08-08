Synchronize branch with remote maintaining context:

1. CHECKPOINT CURRENT STATE:
   ```bash
   git stash save "sync-checkpoint-$(date +%s)"
   ```

2. FETCH AND MERGE:
   ```bash
   git fetch origin
   git merge origin/[branch] --no-edit
   ```

3. RESOLVE CONFLICTS (if any):
   - Maintain sector boundaries
   - Preserve architectural decisions
   - Document resolution in DECISION_CACHE.md

4. RESTORE WORKING STATE:
   ```bash
   git stash pop
   ```

5. UPDATE CONTEXT:
   - Update GIT_CONTEXT.md with new commits
   - Note any architectural changes
   - Reload affected sectors if needed