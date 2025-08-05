# Claude Monitor System - Informe Técnico y Plan de Implementación

## 1. Resumen Ejecutivo

Este documento detalla la arquitectura y el diseño técnico de un sistema de monitoreo de alto rendimiento, diseñado para ejecutarse como un servicio (daemon) en segundo plano en un entorno de Windows Subsystem for Linux (WSL). El sistema tiene dos objetivos principales:

### Objetivos del Sistema

1. **Registro Preciso de Sesiones de Claude**: Implementar la lógica de negocio de las "sesiones" de Claude, que consisten en ventanas de 5 horas que se inician con la primera interacción del usuario, independientemente de la actividad posterior.

2. **Seguimiento de Horas de Trabajo Reales**: Cuantificar el tiempo de uso activo de la herramienta de línea de comandos claude, proporcionando al usuario informes detallados sobre sus horas laborales efectivas.

### Stack Tecnológico

La arquitectura propuesta se basa en una pila tecnológica moderna y de alto rendimiento, seleccionada para garantizar una mínima sobrecarga del sistema, máxima fidelidad de los datos y una base de datos robusta para el análisis:

- **Go (Golang)**: Como lenguaje principal para el desarrollo del daemon de orquestación, elegido por su simplicidad, su potente modelo de concurrencia y su capacidad para compilar binarios estáticos y eficientes.

- **eBPF (extended Berkeley Packet Filter)**: Como motor de captura de datos a nivel de kernel, permitiendo una observabilidad no intrusiva y de alto rendimiento de la actividad de los procesos y la red, sin afectar el rendimiento de la aplicación monitoreada.

- **Kùzu Graph Database**: Como la columna vertebral de persistencia de datos, utilizando un modelo de grafo embebido que representa de forma nativa las complejas interconexiones entre procesos, sesiones y bloques de trabajo.

## 2. Arquitectura General del Sistema

El sistema opera como un pipeline de datos continuo, desde la captura de eventos a nivel de kernel hasta su almacenamiento y posterior análisis.

```
┌─────────────────────────────────────────────────────────────────┐
│                    Claude Monitor Architecture                   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Kernel Layer   │    │   User Space    │    │   Storage       │
│     (eBPF)      │    │   (Go Daemon)   │    │   (Kùzu DB)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ execve/connect  │───▶│ Event Processor │───▶│ Graph Storage   │
│ Syscall Hooks   │    │ Business Logic  │    │ Session/Work    │
│ Ring Buffer     │    │ State Manager   │    │ Relationships   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   CLI Interface │
                       │ Status/Reports  │
                       └─────────────────┘
```

### Capas del Sistema

1. **Capa de Captura (eBPF)**: Programas eBPF de bajo nivel se adjuntan a puntos de anclaje (hooks) específicos en el kernel de Linux para monitorizar eventos clave del sistema de forma pasiva.

2. **Capa de Orquestación (Daemon en Go)**: Un daemon de larga duración escrito en Go carga y gestiona los programas eBPF. Recibe un flujo de eventos del kernel, aplica la lógica de negocio para interpretar las sesiones y las horas de trabajo, y gestiona el estado del sistema.

3. **Capa de Persistencia (Kùzu)**: El daemon de Go se comunica con una base de datos de grafos Kùzu embebida, escribiendo los datos procesados en un fichero local para su almacenamiento persistente.

4. **Capa de Interfaz (CLI en Go)**: El mismo binario de Go proporciona una interfaz de línea de comandos que permite al usuario iniciar el daemon, consultar el estado actual y generar informes históricos consultando la base de datos Kùzu.

## 3. Componente 1: El Agente de Captura eBPF (Los Sentidos)

La base de la recolección de datos es eBPF, elegido por su capacidad para observar el comportamiento del sistema de forma segura y con una sobrecarga de rendimiento casi nula.

### 3.1. Syscalls Monitoreadas

**execve** (y sus variantes clone/fork): Esta es la llamada al sistema que ejecuta un nuevo programa. Al monitorear este evento, podemos detectar de manera fiable cuándo se inicia un proceso con el nombre `claude`. El programa eBPF capturará el Identificador de Proceso (PID) y el nombre del comando.

**connect**: Esta llamada al sistema se utiliza para iniciar una conexión TCP. Dado que la herramienta claude es una CLI, cualquier interacción significativa (enviar un prompt, recibir una respuesta) requerirá una comunicación de red con los servidores de Anthropic. Rastrear la llamada connect desde un proceso claude a los puntos finales de la API de Anthropic es nuestro proxy de alta fidelidad para detectar una "interacción del usuario".

### 3.2. Implementación Técnica (eBPF en C con bpf2go)

El flujo de trabajo recomendado utiliza la herramienta bpf2go de la biblioteca cilium/ebpf, que permite escribir el programa eBPF en C y generar automáticamente el código Go necesario para cargarlo y gestionarlo.

**claude_tracker.c** (Programa eBPF conceptual):

```c
#include <vmlinux.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

// Estructura para enviar eventos a espacio de usuario
struct event {
    u32 pid;
    char comm[16];
    u32 event_type; // 1 for exec, 2 for connect
};

// Mapa de Ring Buffer para comunicación con Go
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

// Hook para la llamada al sistema execve
SEC("tp/syscalls/sys_enter_execve")
int handle_execve(struct trace_event_raw_sys_enter *ctx) {
    struct event *e;
    e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
    if (!e) {
        return 0;
    }

    e->pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&e->comm, sizeof(e->comm));
    e->event_type = 1; // Tipo de evento: exec

    bpf_ringbuf_submit(e, 0);
    return 0;
}

// Hook para la llamada al sistema connect
SEC("tp/syscalls/sys_enter_connect")
int handle_connect(struct trace_event_raw_sys_enter *ctx) {
    struct event *e;
    e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
    if (!e) {
        return 0;
    }
    
    e->pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&e->comm, sizeof(e->comm));
    e->event_type = 2; // Tipo de evento: connect

    bpf_ringbuf_submit(e, 0);
    return 0;
}

char LICENSE SEC("license") = "GPL";
```

## 4. Componente 2: El Daemon de Orquestación en Go (El Cerebro)

El daemon es el núcleo lógico del sistema. Escrito en Go, es responsable de gestionar el ciclo de vida de los programas eBPF, procesar los eventos entrantes y aplicar las reglas de negocio para las sesiones y las horas de trabajo.

### 4.1. Estructura del Daemon

El daemon se ejecutará como un único proceso en segundo plano. Utilizará goroutines para manejar tareas concurrentes de manera eficiente:

- Una goroutine principal para la inicialización y la gestión de señales de apagado
- Una goroutine para leer y procesar eventos del ringbuf de eBPF
- Una goroutine para escanear periódicamente los procesos del sistema y mantener una lista actualizada de los PIDs de claude

### 4.2. Lógica de Estado y Reglas de Negocio

El daemon mantendrá el estado actual en memoria y lo persistirá en Kùzu.

#### 4.2.1. Registro de Sesiones de Claude

La lógica para las sesiones es estrictamente basada en el tiempo y se activa por la primera interacción detectada.

**Estado Requerido**: `currentSessionEndTime` (timestamp)

**Flujo Lógico**:
1. El daemon detecta un evento connect de un proceso claude a un punto final de la API de Anthropic
2. Obtiene la hora actual: `now = time.Now()`
3. Comprueba si hay una sesión activa: `if now > currentSessionEndTime`
4. Si no hay sesión activa (la condición es verdadera):
   - Se registra el inicio de una nueva sesión
   - Se actualiza el estado: `currentSessionEndTime = now.Add(5 * time.Hour)`
   - Se crea un nuevo nodo Session en la base de datos Kùzu
5. Si hay una sesión activa (la condición es falsa):
   - La interacción se considera parte de la sesión existente
   - No se realiza ninguna acción sobre el estado de la sesión

#### 4.2.2. Seguimiento de Horas de Trabajo

La lógica para las horas de trabajo se basa en la actividad continua, con un umbral de inactividad para definir los bloques de trabajo.

**Estado Requerido**: `currentWorkBlockStartTime` (timestamp), `lastActivityTime` (timestamp)  
**Constante**: `WORK_BLOCK_TIMEOUT = 5 * time.Minute`

**Flujo Lógico**:
1. El daemon detecta un evento connect de un proceso claude (una "interacción")
2. Obtiene la hora actual: `now = time.Now()`
3. Comprueba si es el inicio de un nuevo bloque de trabajo: `if now.Sub(lastActivityTime) > WORK_BLOCK_TIMEOUT`
4. Si es un nuevo bloque de trabajo (la condición es verdadera):
   - Si `currentWorkBlockStartTime` no es nulo, significa que el bloque de trabajo anterior ha finalizado
   - Se calcula su duración `lastActivityTime.Sub(currentWorkBlockStartTime)` y se persiste un nodo WorkBlock en Kùzu, relacionándolo con la sesión actual
   - Se inicia un nuevo bloque: `currentWorkBlockStartTime = now`
5. En todos los casos:
   - Se actualiza la última actividad: `lastActivityTime = now`
6. Al apagar el daemon de forma controlada, se registrará el bloque de trabajo final

## 5. Componente 3: Persistencia con Kùzu (La Memoria)

Para el almacenamiento de datos, se ha elegido Kùzu, una base de datos de grafos embebida. Este modelo es conceptualmente superior para este caso de uso, ya que los eventos del sistema son inherentemente un grafo de entidades interconectadas.

### 5.1. Diseño del Esquema del Grafo

El esquema se definirá utilizando el DDL de Cypher para garantizar la integridad de los datos:

```cypher
-- Definición de Nodos (Entidades)
CREATE NODE TABLE Session(
    sessionID STRING,
    startTime TIMESTAMP,
    endTime TIMESTAMP,
    PRIMARY KEY (sessionID)
);

CREATE NODE TABLE WorkBlock(
    blockID STRING,
    startTime TIMESTAMP,
    endTime TIMESTAMP,
    durationSeconds INT64,
    PRIMARY KEY (blockID)
);

CREATE NODE TABLE Process(
    PID INT64,
    command STRING,
    startTime TIMESTAMP,
    PRIMARY KEY (startTime)
);

-- Definición de Relaciones (Conexiones)
CREATE REL TABLE EXECUTED_DURING(
    FROM Process TO Session
);

CREATE REL TABLE CONTAINS(
    FROM Session TO WorkBlock
);
```

### 5.2. Interacción con la Base de Datos desde Go

El daemon utilizará la biblioteca oficial de Kùzu para Go para interactuar con el fichero de la base de datos.

Ejemplo de inserción de datos (conceptual):

```go
package main

import (
    "github.com/kuzudb/go-kuzu"
    "fmt"
    "time"
)

func logNewSession(conn *kuzu.Connection, sessionID string, startTime time.Time) error {
    query := `CREATE (s:Session {
        sessionID: $sessionID, 
        startTime: $startTime, 
        endTime: $endTime
    })`
    
    params := map[string]interface{}{
        "sessionID": sessionID,
        "startTime": startTime,
        "endTime":   startTime.Add(5 * time.Hour),
    }

    _, err := conn.Query(query, params)
    return err
}
```

## 6. Componente 4: Interfaz de Usuario y Reportes (La Voz)

Toda la interacción del usuario se realizará a través de la línea de comandos. El binario compilado (claude-monitor) actuará como el punto de entrada para todas las operaciones.

### 6.1. Comandos Disponibles

**`sudo ./claude-monitor start`**: Inicia el daemon de monitoreo en segundo plano. Requerirá sudo para cargar los programas eBPF.

**`./claude-monitor status`**: Consulta la base de datos y muestra el estado actual.

Salida de ejemplo si no hay sesión activa:
```
Estado de la Sesión de Claude: Inactiva
Horas de Trabajo Hoy: 2h 15m
```

Salida de ejemplo si hay una sesión activa:
```
Estado de la Sesión de Claude: Activa
La sesión finaliza a las: 19:45
Horas de Trabajo Hoy: 3h 30m
```

**`./claude-monitor report [--period=daily|weekly|monthly]`**: Genera un informe agregado del uso.

Salida de ejemplo para `report --period=weekly`:
```
--- Informe Semanal de Uso de Claude ---
Período: 2025-08-04 a 2025-08-10

Sesiones Utilizadas: 8

Horas de Trabajo por Día:
- Lunes:    4h 30m
- Martes:   5h 15m
- Miércoles: 2h 05m
- Jueves:   6h 45m
- Viernes:  3h 10m

Total Horas de Trabajo Semanales: 21h 45m
```

### 6.2. Consultas Cypher para Reportes

Los informes se generan ejecutando consultas Cypher sobre la base de datos Kùzu.

Consulta para el total de horas de trabajo diarias:
```cypher
MATCH (wb:WorkBlock)
WHERE wb.startTime >= date_trunc('day', now())
RETURN sum(wb.durationSeconds)
```

Consulta para el número de sesiones en la última semana:
```cypher
MATCH (s:Session)
WHERE s.startTime >= now() - interval '7 days'
RETURN count(s)
```

## 7. Plan de Implementación

### 7.1. Fase 1: Arquitectura Base y eBPF
**Agente Responsable**: `architecture-designer` + `ebpf-specialist`
**Duración Estimada**: 2-3 semanas

**Tareas**:
- [ ] Configurar estructura de proyecto Go con módulos
- [ ] Implementar programas eBPF básicos (execve, connect)
- [ ] Configurar bpf2go para generación automática de código
- [ ] Implementar ring buffer y comunicación kernel-userspace
- [ ] Pruebas básicas de captura de eventos

**Entregables**:
- Programas eBPF funcionales
- Integración Go-eBPF básica
- Captura de eventos claude

### 7.2. Fase 2: Daemon Core y Lógica de Negocio
**Agente Responsable**: `daemon-core`
**Duración Estimada**: 2-3 semanas

**Tareas**:
- [ ] Implementar estructura del daemon principal
- [ ] Desarrollar lógica de sesiones (5 horas)
- [ ] Implementar seguimiento de work blocks (5 min timeout)
- [ ] Gestión de estado y concurrencia
- [ ] Manejo de señales y shutdown graceful

**Entregables**:
- Daemon funcional con lógica de negocio
- Gestión correcta de sesiones y work blocks
- Tests unitarios para lógica crítica

### 7.3. Fase 3: Persistencia y Base de Datos
**Agente Responsable**: `database-manager`
**Duración Estimada**: 2 semanas

**Tareas**:
- [ ] Configurar Kùzu database embebida
- [ ] Implementar esquema de grafos
- [ ] Desarrollar repositorios para Session y WorkBlock
- [ ] Implementar transacciones y manejo de errores
- [ ] Queries para reportes básicos

**Entregables**:
- Base de datos funcional
- Operaciones CRUD completas
- Queries de reporting optimizadas

### 7.4. Fase 4: CLI e Interfaz de Usuario
**Agente Responsable**: `cli-interface`
**Duración Estimada**: 2 semanas

**Tareas**:
- [ ] Implementar comandos CLI con Cobra
- [ ] Desarrollar comando `start` con verificación de privilegios
- [ ] Implementar comando `status` con formateo
- [ ] Desarrollar sistema de reportes con múltiples formatos
- [ ] Configuración y validación

**Entregables**:
- CLI completa y funcional
- Documentación de comandos
- Sistema de reportes robusto

### 7.5. Fase 5: Testing e Integración
**Agente Responsable**: Todos los agentes
**Duración Estimada**: 1-2 semanas

**Tareas**:
- [ ] Tests de integración end-to-end
- [ ] Pruebas de rendimiento y carga
- [ ] Validación en entorno WSL
- [ ] Documentación completa
- [ ] Optimizaciones finales

**Entregables**:
- Sistema completamente funcional
- Suite de tests completa
- Documentación de usuario

## 8. Despliegue y Ejecución en WSL

El sistema está diseñado para ser autocontenido y fácil de desplegar en un entorno WSL.

### 8.1. Compilación
El proyecto Go se compila en un único binario estático:
```bash
go build -o claude-monitor
```

### 8.2. Ejecución del Daemon
Para iniciar el monitoreo, el usuario ejecutará el binario con sudo:
```bash
sudo ./claude-monitor start
```

El daemon se desvinculará de la terminal y continuará ejecutándose. Gestionará su propio PID para evitar múltiples instancias.

### 8.3. Permisos
La ejecución como root es un requisito indispensable, ya que las operaciones de eBPF (cargar programas en el kernel, adjuntar a tracepoints) requieren privilegios de superusuario.

## 9. Consideraciones Técnicas

### 9.1. Seguridad
- Validación de entrada en todos los puntos
- Manejo seguro de privilegios root
- Logging de eventos de seguridad
- Protección contra inyección en queries

### 9.2. Rendimiento
- Overhead mínimo del sistema (<1% CPU)
- Uso eficiente de memoria
- Optimización de queries de base de datos
- Filtrado a nivel de kernel para reducir eventos

### 9.3. Fiabilidad
- Recuperación automática de errores
- Persistencia de estado crítico
- Manejo robusto de fallos de red
- Logs detallados para debugging

### 9.4. Mantenibilidad
- Código modular y bien documentado
- Tests automatizados
- Monitoreo de métricas internas
- Documentación técnica completa

## 10. Conclusión

La arquitectura propuesta presenta una solución robusta, eficiente y técnicamente avanzada para el monitoreo del uso de la herramienta claude. La combinación de Go para una lógica de aplicación concurrente y mantenible, eBPF para una captura de datos a nivel de kernel precisa y de bajo impacto, y Kùzu para una persistencia de datos conceptualmente alineada y de alto rendimiento, da como resultado un sistema que cumple con todos los requisitos del usuario.

El diseño del daemon garantiza un funcionamiento automático y continuo, mientras que la interfaz de línea de comandos proporciona al usuario un control total y acceso a informes valiosos sobre sus patrones de trabajo y el uso de las sesiones de Claude.

### Próximos Pasos

1. **Revisar y aprobar el plan de implementación**
2. **Configurar el entorno de desarrollo**
3. **Comenzar con la Fase 1: Arquitectura Base y eBPF**
4. **Establecer reuniones de revisión semanales**
5. **Definir métricas de éxito para cada fase**

---

**Documento de Referencia**: Este archivo debe ser utilizado como guía principal para la implementación del sistema Claude Monitor. Consultar con los agentes especializados correspondientes para cada fase del desarrollo.