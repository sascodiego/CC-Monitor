Switch sectors with context preservation:

1. CHECKPOINT CURRENT SECTOR:
   ```bash
   # Save current work
   git add -A
   git commit -m "feat([current-sector]): [completed work]"
   
   # Tag for easy return
   git tag checkpoint-[sector]-[timestamp]
   ```

2. CONTEXT TRANSITION:
   ```markdown
   CLOSING SECTOR: [current]
   - Files modified: [list]
   - Patterns established: [list]
   - Save to DECISION_CACHE.md
   
   OPENING SECTOR: [new]
   - Load sector rules
   - Identify interfaces
   - Check dependencies
   ```

3. TOKEN OPTIMIZATION:
   If tokens > 50%:
     - Run /compact-smart before switch
     - Preserve new sector in compact
   
4. UPDATE BOUNDARIES:
   ```markdown
   ACTIVE_SECTOR: [new]
   LOCKED_SECTORS: [previous]
   FORBIDDEN: Cross-sector changes
   ```

5. CONTINUE IN NEW SECTOR:
   Load only new sector files