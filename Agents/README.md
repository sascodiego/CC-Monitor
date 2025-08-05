# Claude Monitor - Specialized Agent System

Este directorio contiene agentes especializados para el desarrollo del sistema de monitoreo Claude. Cada agente es un experto en un dominio específico del proyecto y debe ser utilizado cuando se trabaje en su área de especialización.

## 🏗️ Arquitectura de Agentes

### **architecture-designer** - Arquitecto del Sistema
**Especialización**: Diseño de arquitectura general, patrones Go, interfaces, integración eBPF/Go/Kùzu

**Usar cuando necesites**:
- Diseñar la arquitectura general del sistema
- Definir interfaces y contratos entre componentes
- Establecer patrones de inyección de dependencias
- Coordinar la integración entre eBPF, Go y Kùzu
- Resolver problemas de diseño arquitectónico

**Ejemplos de uso**:
- "Necesito diseñar la arquitectura del sistema de monitoreo"
- "¿Cómo debo estructurar las interfaces entre componentes?"
- "Ayúdame a implementar patrones de DI en Go"

### **ebpf-specialist** - Especialista en eBPF
**Especialización**: Programación eBPF, syscalls, comunicación kernel-userspace, optimización de rendimiento

**Usar cuando necesites**:
- Escribir programas eBPF en C
- Implementar monitoreo de syscalls (execve, connect)
- Optimizar comunicación ring buffer
- Manejar recursos del kernel
- Debuggear programas eBPF

**Ejemplos de uso**:
- "Necesito crear programas eBPF para monitorear procesos claude"
- "¿Cómo optimizo el rendimiento de los ring buffers?"
- "Ayúdame a implementar filtrado a nivel de kernel"

### **daemon-core** - Orquestador del Daemon
**Especialización**: Lógica de negocio Go, gestión de sesiones, work blocks, concurrencia, ciclo de vida del daemon

**Usar cuando necesites**:
- Implementar lógica de sesiones (ventanas de 5 horas)
- Manejar work blocks (timeout de 5 minutos)
- Coordinar goroutines y estado compartido
- Gestionar el ciclo de vida del daemon
- Implementar patrones de concurrencia

**Ejemplos de uso**:
- "Necesito implementar la lógica de sesiones de Claude"
- "¿Cómo manejo la concurrencia entre múltiples goroutines?"
- "Ayúdame a implementar el shutdown graceful del daemon"

### **database-manager** - Gestor de Base de Datos
**Especialización**: Operaciones Kùzu, queries Cypher, schema de grafos, transacciones, optimización de rendimiento

**Usar cuando necesites**:
- Diseñar schema de base de datos en Kùzu
- Escribir queries Cypher eficientes
- Implementar transacciones y manejo de conexiones
- Optimizar rendimiento de consultas
- Generar reportes complejos

**Ejemplos de uso**:
- "Necesito diseñar el schema de grafos para sesiones y work blocks"
- "¿Cómo optimizo las consultas de reporting?"
- "Ayúdame a implementar transacciones ACID en Kùzu"

### **cli-interface** - Interfaz de Usuario CLI
**Especialización**: Comandos CLI, parsing de argumentos, formateo de salida, experiencia de usuario

**Usar cuando necesites**:
- Diseñar comandos CLI intuitivos
- Implementar parsing de argumentos
- Crear outputs formateados y legibles
- Implementar features interactivos
- Mejorar la experiencia del usuario

**Ejemplos de uso**:
- "Necesito crear comandos CLI para controlar el daemon"
- "¿Cómo mejoro el formateo de los reportes?"
- "Ayúdame a implementar features interactivos"

## 🎯 Cómo Usar los Agentes

### **1. Identificar el Dominio**
Antes de solicitar ayuda, identifica qué dominio del sistema necesitas trabajar:
- ¿Es arquitectura general? → `architecture-designer`
- ¿Es programación a nivel kernel? → `ebpf-specialist`
- ¿Es lógica de negocio del daemon? → `daemon-core`
- ¿Es persistencia de datos? → `database-manager`
- ¿Es interfaz de usuario? → `cli-interface`

### **2. Usar la Sintaxis Correcta**
```
Necesito usar el agente [nombre-agente] para [descripción de la tarea]
```

Ejemplo:
```
Necesito usar el agente ebpf-specialist para implementar el monitoreo de syscalls connect
```

### **3. Proporcionar Contexto**
Siempre incluye:
- **Qué** estás intentando lograr
- **Por qué** necesitas hacerlo
- **Contexto** del sistema o componente
- **Restricciones** o requerimientos específicos

### **4. Ejemplo de Sesión Completa**
```
Usuario: "Necesito usar el agente daemon-core para implementar la lógica de sesiones que duran exactamente 5 horas desde la primera interacción"

Asistente: "Usaré el agente daemon-core para implementar la lógica de gestión de sesiones con ventanas de 5 horas..."

[El agente implementa la solución específica con patrones de concurrencia Go, 
gestión de estado, y lógica de timing precisa]
```

## 🔄 Coordinación Entre Agentes

Los agentes están diseñados para trabajar juntos de manera coordinada:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ architecture-   │───▶│ ebpf-specialist │───▶│ daemon-core     │
│ designer        │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ database-       │◀───┤ cli-interface   │───▶│ [Otros agentes] │
│ manager         │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### **Flujo de Trabajo Típico**:
1. **architecture-designer** define interfaces y patrones
2. **ebpf-specialist** implementa captura de eventos
3. **daemon-core** procesa eventos y aplica lógica de negocio
4. **database-manager** persiste datos y genera reportes
5. **cli-interface** proporciona acceso al usuario

## 🛡️ Estándares de Calidad

Todos los agentes siguen el **estándar obligatorio de comentarios**:

```go
/**
 * AGENT:     [nombre-del-agente]
 * TRACE:     [ID del ticket/issue, ej: CLAUDE-123]
 * CONTEXT:   [Descripción del propósito y contexto]
 * REASON:    [Razón detrás de esta implementación]
 * CHANGE:    [Descripción del cambio o "Initial implementation."]
 * PREVENTION:[Consideraciones para prevenir problemas futuros]
 * RISK:      [Nivel de riesgo y consecuencias si PREVENTION falla]
 */
```

## 📚 Recursos Adicionales

- **CLAUDE.md**: Prompt de sistema principal con contexto del proyecto
- **PROYECT.md**: Documentación técnica detallada del sistema
- **Cada agent.md**: Documentación específica de cada agente

## ⚠️ Mejores Prácticas

1. **Un agente por problema**: No mezcles dominios en una sola solicitud
2. **Contexto específico**: Proporciona detalles relevantes al dominio del agente
3. **Seguimiento**: Usa los mismos agentes para seguimiento y refinamiento
4. **Coordinación**: Considera cómo tu trabajo afecta otros componentes
5. **Documentación**: Mantén actualizados los comentarios y documentación

---

**Nota**: Los agentes están diseñados para ser especialistas en sus dominios. Usar el agente correcto para cada tarea garantiza soluciones optimales y código de alta calidad.