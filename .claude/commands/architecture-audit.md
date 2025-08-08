Run comprehensive SOLID/Clean Code audit:

1. ENABLE GLOBAL MODE:
   /global-mode "Architecture audit"

2. ANALYZE STRUCTURE:
   ```
   Check for:
   - Single Responsibility violations
   - Open/Closed principle compliance
   - Liskov Substitution issues
   - Interface Segregation problems
   - Dependency Inversion violations
   ```

3. SCAN PATTERNS:
   - Identify design patterns used
   - Find anti-patterns
   - Check coupling/cohesion
   - Validate boundaries

4. GENERATE REPORT:
   Save to docs/AUDIT_RESULTS.md:
   ```markdown
   # Architecture Audit - [date]
   
   ## SOLID Compliance
   - SRP: [score and violations]
   - OCP: [score and violations]
   - LSP: [score and violations]
   - ISP: [score and violations]
   - DIP: [score and violations]
   
   ## Recommendations
   [Prioritized list of fixes]
   
   ## Implementation Plan
   [Sector-by-sector fix plan]
   ```

5. RETURN TO SECTORIAL:
   "Audit complete. Returning to sectorial mode for implementation."