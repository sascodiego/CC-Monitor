# Claude Monitor System - Arquitectura Técnica
# Sistema de Monitoreo con Go + KuzuDB

## 1. Resumen Ejecutivo

Este documento detalla la arquitectura y el diseño técnico de Claude Monitor, un sistema de monitoreo de actividad para Claude Code basado en Go y KuzuDB. Diseñado para ejecutarse como un servicio (daemon) en segundo plano en un entorno de Windows Subsystem for Linux (WSL), el sistema utiliza hooks de Claude Code para detectar actividad y realizar seguimiento preciso de horarios de trabajo.

### Objetivos del Sistema

1. **Registro Preciso de Sesiones de Claude**: Implementar la lógica de negocio de las "sesiones" de Claude, que consisten en ventanas de 5 horas que se inician con la primera interacción del usuario, independientemente de la actividad posterior.

2. **Seguimiento de Horas de Trabajo Reales**: Cuantificar el tiempo de uso activo de la herramienta de línea de comandos claude, proporcionando al usuario informes detallados sobre sus horas laborales efectivas.

### Stack Tecnológico - Arquitectura Go + KuzuDB

Claude Monitor utiliza un stack tecnológico simple y confiable:

- **Go Language**: Desarrollo rápido y confiable con excelente tooling
- **KuzuDB Graph Database**: Base de datos de grafos para relaciones complejas
- **Claude Code Hooks**: Sistema de detección de actividad mediante hooks configurados
- **CLI con Cobra**: Interfaz de línea de comandos user-friendly

### Ventajas de la Arquitectura Go + KuzuDB

| Aspecto | Especificación | Beneficio |
|---------|----------------|-----------|
| Desarrollo | Rápido y simple | Time-to-market reducido |
| Detección de actividad | Hook-based | Precisión del 100% |
| Memoria | < 100MB RSS | Impacto mínimo en sistema |
| CPU | < 2% average | Casi imperceptible |
| Mantenimiento | Go estándar | Fácil de mantener |
| Datos | Graph database | Consultas relacionales complejas |

## 2. Arquitectura del Sistema

```
┌───────────────────────────────────────────────────────────────┐
│             Claude Monitor Architecture (Go + KuzuDB)           │
└───────────────────────────────────────────────────────────────┘

┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Claude Code    │    │   Go Daemon     │    │   KuzuDB        │
│   (Hooks)        │    │  (Processor)    │    │  (Storage)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Hook Execution  │───▶│ Session Manager │───▶│ Graph Relations │
│ "claude-code    │    │ Work Block      │    │ Session Data    │
│ action"         │    │ Tracker         │    │ Project Info    │
│ Command         │    │ Timer Logic     │    │ Time Analytics  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   CLI Interface │
                       │   (Go + Cobra)  │
                       └─────────────────┘
```

## 3. Componente 1: Sistema de Detección por Hooks de Claude Code

### 3.1. Sistema de Hooks de Claude Code

Claude Monitor utiliza el sistema de hooks de Claude Code para detección precisa:
- **Hook Configuration**: Comando ejecutado antes de cada acción de Claude
- **Precision**: 100% de precisión en detección de actividad
- **Project Detection**: Identificación automática del proyecto actual
- **Timestamp Accuracy**: Registro exacto de tiempo de actividad

### 3.2. Configuración del Hook de Claude Code

**Claude Code Hook Configuration:**

```bash
# Configuración del hook en Claude Code
# Este comando se ejecuta antes de cada acción de Claude
claude-code action
```

**Estructura del Comando Hook:**

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    "encoding/json"
    "net/http"
    "bytes"
)

type ActivityEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    ProjectPath string    `json:"project_path"`
    ProjectName string    `json:"project_name"`
    WorkingDir  string    `json:"working_dir"`
    UserID      string    `json:"user_id"`
    SessionID   string    `json:"session_id"`
    EventType   string    `json:"event_type"`
}

func main() {
    // Detectar proyecto actual
    workingDir, _ := os.Getwd()
    projectName := filepath.Base(workingDir)
    
    // Crear evento de actividad
    event := ActivityEvent{
        Timestamp:   time.Now(),
        ProjectPath: workingDir,
        ProjectName: projectName,
        WorkingDir:  workingDir,
        UserID:      os.Getenv("USER"),
        SessionID:   generateSessionID(),
        EventType:   "claude_action",
    }
    
    // Enviar al daemon local
    sendToDaemon(event)
}

func sendToDaemon(event ActivityEvent) {
    jsonData, err := json.Marshal(event)
    if err != nil {
        return
    }
    
    // Enviar al daemon local via HTTP
    resp, err := http.Post("http://localhost:8080/activity", 
                          "application/json", 
                          bytes.NewBuffer(jsonData))
    if err != nil {
        // Fallback: escribir a archivo local
        writeToLocalFile(event)
        return
    }
    defer resp.Body.Close()
}

func writeToLocalFile(event ActivityEvent) {
    homeDir, _ := os.UserHomeDir()
    logFile := filepath.Join(homeDir, ".claude-monitor", "activity.log")
    
    file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return
    }
    defer file.Close()
    
    jsonData, _ := json.Marshal(event)
    file.WriteString(string(jsonData) + "\n")
}

func generateSessionID() string {
    // Lógica para determinar sesión de 5 horas
    return fmt.Sprintf("session_%d", time.Now().Unix())
}
```

### 3.3. Daemon Go para Procesamiento de Eventos

**Daemon HTTP Server en Go:**

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "time"
    "context"
    "github.com/gorilla/mux"
)

type ClaudeMonitorDaemon struct {
    sessionManager *SessionManager
    workTracker    *WorkBlockTracker
    database       *KuzuDBConnection
}

type ActivityEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    ProjectPath string    `json:"project_path"`
    ProjectName string    `json:"project_name"`
    WorkingDir  string    `json:"working_dir"`
    UserID      string    `json:"user_id"`
    SessionID   string    `json:"session_id"`
    EventType   string    `json:"event_type"`
}

func (d *ClaudeMonitorDaemon) handleActivity(w http.ResponseWriter, r *http.Request) {
    var event ActivityEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Procesar evento de actividad
    d.processActivityEvent(event)
    
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Event processed"))
}

func (d *ClaudeMonitorDaemon) processActivityEvent(event ActivityEvent) {
    // Determinar sesión activa (ventana de 5 horas)
    session := d.sessionManager.GetOrCreateSession(event.Timestamp)
    
    // Actualizar bloque de trabajo (timeout de 5 minutos)
    workBlock := d.workTracker.UpdateWorkBlock(session.ID, event.Timestamp, event.ProjectName)
    
    // Guardar en base de datos
    d.database.SaveActivity(session, workBlock, event)
}

func main() {
    daemon := &ClaudeMonitorDaemon{
        sessionManager: NewSessionManager(),
        workTracker:    NewWorkBlockTracker(),
        database:       NewKuzuDBConnection(),
    }
    
    router := mux.NewRouter()
    router.HandleFunc("/activity", daemon.handleActivity).Methods("POST")
    
    log.Println("Claude Monitor Daemon started on :8080")
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

## 4. Plan de Implementación Go + KuzuDB

### Fase 1: Hook Integration Setup (1-2 semanas)
**Objetivo**: Configurar sistema de hooks de Claude Code

**Tareas**:
- [ ] Crear comando "claude-code action" en Go
- [ ] Configurar hook en Claude Code settings
- [ ] Implementar detección de proyecto actual
- [ ] Setup daemon HTTP server básico
- [ ] Testing de hook execution

**Métricas de Éxito**:
- Hook se ejecuta en cada acción de Claude
- Detección correcta de proyecto
- Comunicación exitosa con daemon

### Fase 2: Core Business Logic (2-3 semanas)
**Objetivo**: Implementar lógica de sesiones y bloques de trabajo

**Tareas**:
- [ ] Session Manager con ventanas de 5 horas
- [ ] Work Block Tracker con timeout de 5 minutos
- [ ] Timer logic para inicio/fin de trabajo
- [ ] Event processing pipeline
- [ ] Memory management y persistence

**Entregables**:
- Sesiones funcionando correctamente
- Work blocks con detección de idle
- Cálculo de horas reales vs horas totales

### Fase 3: KuzuDB Integration (2-3 semanas)
**Objetivo**: Integrar base de datos de grafos

**Tareas**:
- [ ] Setup KuzuDB con Go driver
- [ ] Diseñar schema de grafos (Sessions, WorkBlocks, Projects)
- [ ] Implementar repository patterns
- [ ] Query optimization para reportes
- [ ] Data persistence y recovery

**Beneficios Esperados**:
- Consultas relacionales complejas
- Análisis de patrones de trabajo
- Reportes detallados por proyecto

### Fase 4: CLI User Interface (1-2 semanas)
**Objetivo**: Interfaz CLI user-friendly

**Tareas**:
- [ ] CLI con Cobra framework
- [ ] Comandos para ver actividad diaria/semanal/mensual
- [ ] Reportes con formato atractivo
- [ ] Export a diferentes formatos
- [ ] Shell completions

### Fase 5: Production Deployment (1 semana)
**Objetivo**: Deploy y documentación

**Tareas**:
- [ ] Build system y distribución
- [ ] Documentación de usuario
- [ ] Scripts de instalación
- [ ] Testing de integración completo
- [ ] Launch preparation

## 5. Componentes del Sistema Go

### Hook Command ("claude-code action")
**Responsabilidad**: Detección de actividad de Claude Code
- Ejecutarse antes de cada acción de Claude
- Detectar proyecto actual automáticamente
- Enviar eventos al daemon via HTTP
- Fallback a archivo local si daemon no disponible

### Go Daemon (HTTP Server)
**Responsabilidad**: Procesamiento central de eventos
- Recibir eventos de actividad via HTTP API
- Implementar lógica de sesiones de 5 horas
- Gestionar bloques de trabajo con timeout de 5 minutos
- Persistir datos en KuzuDB

### Session Manager
**Responsabilidad**: Gestión de sesiones de Claude
- Crear sesiones que duran exactamente 5 horas
- Determinar cuando iniciar nueva sesión
- Tracking de primera interacción
- Manejo de overlapping entre sesiones

### Work Block Tracker
**Responsabilidad**: Seguimiento de bloques de trabajo activo
- Detectar inicio de trabajo (primera actividad)
- Detectar fin de trabajo (5 minutos sin actividad)
- Calcular duración real de trabajo
- Asociar bloques con proyectos específicos

## 6. Funcionalidades del Sistema

### Datos Core Recolectados
- **Timestamp exacto** de cada actividad de Claude
- **Proyecto activo** donde se realiza la actividad
- **Duración real** de trabajo (tiempo activo vs idle)
- **Horarios de trabajo** (hora inicio y fin de trabajo)
- **Sesiones de Claude** (ventanas de 5 horas)

### Reportes Disponibles
- **Vista diaria**: Actividad del día actual
- **Vista semanal**: Resumen de la semana actual
- **Vista mensual**: Actividad del mes actual
- **Historial**: Consulta de meses anteriores
- **Por proyecto**: Análisis de tiempo por proyecto específico

## 7. Arquitectura de Datos (KuzuDB)

### Schema de Grafos
```cypher
// Nodos
CREATE NODE TABLE User(id STRING, name STRING, PRIMARY KEY(id));
CREATE NODE TABLE Project(id STRING, name STRING, path STRING, PRIMARY KEY(id));
CREATE NODE TABLE Session(id STRING, start_time TIMESTAMP, end_time TIMESTAMP, PRIMARY KEY(id));
CREATE NODE TABLE WorkBlock(id STRING, start_time TIMESTAMP, end_time TIMESTAMP, duration_seconds INT, PRIMARY KEY(id));

// Relaciones
CREATE REL TABLE WORKS_ON(FROM User TO Project);
CREATE REL TABLE HAS_SESSION(FROM User TO Session);
CREATE REL TABLE CONTAINS_WORK(FROM Session TO WorkBlock);
CREATE REL TABLE WORK_IN_PROJECT(FROM WorkBlock TO Project);
```

### Consultas de Ejemplo
```cypher
// Horas trabajadas hoy
MATCH (u:User)-[:HAS_SESSION]->(s:Session)-[:CONTAINS_WORK]->(w:WorkBlock)
WHERE s.start_time >= today()
RETURN SUM(w.duration_seconds) / 3600 as hours_today;

// Actividad por proyecto esta semana
MATCH (p:Project)<-[:WORK_IN_PROJECT]-(w:WorkBlock)
WHERE w.start_time >= startOfWeek()
RETURN p.name, SUM(w.duration_seconds) / 3600 as hours;
```

## 8. Interfaz CLI User-Friendly

### Comandos Principales
```bash
# Ver actividad de hoy
claude-monitor today

# Ver actividad de la semana
claude-monitor week

# Ver actividad del mes
claude-monitor month

# Ver mes específico
claude-monitor month --month=2024-12

# Ver por proyecto
claude-monitor project --name="Mi Proyecto"

# Status del daemon
claude-monitor status

# Iniciar/parar daemon
claude-monitor start
claude-monitor stop
```

### Output Example
```
📅 Actividad de Hoy - 2024-08-05
┌──────────────────────────────────────────────────┐
│ 🕰️  Horario de Trabajo: 09:15 - 17:30 (8h 15m)     │
│ ⏱️  Tiempo Activo: 6h 45m                          │
│ ⏸️  Tiempo Idle: 1h 30m                           │
└──────────────────────────────────────────────────┘

📁 Actividad por Proyecto:
• Claude-Monitor        4h 30m  (66.7%)
• Documentation        1h 45m  (25.9%)
• Code Review          30m     (7.4%)

📊 Bloques de Trabajo:
09:15-11:30  Claude-Monitor    (2h 15m)
11:45-13:00  Documentation     (1h 15m)
14:00-16:15  Claude-Monitor    (2h 15m)
16:15-16:45  Code Review       (30m)
16:45-17:30  Documentation     (45m)
```

## 9. Conclusión

El sistema Claude Monitor con arquitectura Go + KuzuDB + Hooks proporciona:

1. **Detección Precisa**: 100% de precisión usando hooks de Claude Code
2. **Datos Ricos**: Información detallada de proyectos y patrones de trabajo
3. **User Experience**: CLI intuitiva con reportes atractivos
4. **Flexibilidad**: Consultas complejas gracias a KuzuDB
5. **Simplicidad**: Arquitectura simple y mantenible en Go

### Capacidades del Sistema

- **Seguimiento de Sesiones**: Ventanas de 5 horas desde primera interacción
- **Detección de Trabajo Activo**: Bloques de trabajo con timeout de 5 minutos
- **Análisis por Proyecto**: Tiempo dedicado a cada proyecto automáticamente
- **Métricas Duales**: Horas reales trabajadas + horario de inicio/fin
- **Reportes Flexibles**: Vista diaria, semanal, mensual e histórica

### Próximos Pasos

1. **Implementar hook "claude-code action"**
2. **Desarrollar daemon Go con HTTP API**
3. **Integrar KuzuDB para persistencia**
4. **Crear CLI con reportes user-friendly**
5. **Testing y deployment**

---

**Documento de Referencia**: Este archivo define la estrategia de migración hacia Rust con enfoque eBPF-First. Consultar con los agentes especializados de Rust para cada fase del desarrollo.