Initialize new project with proper structure:

1. ENABLE GLOBAL MODE:
   "Initializing project requires global access"

2. CREATE STRUCTURE:
   ```
   src/
   ├── domain/         # Business logic
   ├── application/    # Use cases
   ├── infrastructure/ # External interfaces
   ├── shared/        # Utilities
   └── tests/         # Test files
   
   .claude/
   ├── commands/      # Command files
   ├── context/       # Context files
   └── CLAUDE.md      # Project rules
   
   docs/
   └── README.md      # Documentation
   ```

3. DEFINE SECTORS:
   Save to SECTOR_PLAN.md:
   ```markdown
   # Sector Definitions
   
   SECTOR: domain
   - Business entities
   - Domain services
   - Value objects
   
   SECTOR: application
   - Use cases
   - Application services
   - DTOs
   
   [etc...]
   ```

4. INITIALIZE GIT:
   ```bash
   git init
   git add .
   git commit -m "feat: Initialize project structure"
   ```

5. RETURN TO SECTORIAL:
   "Project initialized. Ready for sectorial development."