# Bot de Mesas de Examen - UNNE

Este proyecto es una aplicación web en **Go (Golang)** que permite gestionar mesas de examen y ofrece un **Chatbot** para que los alumnos consulten fechas y horarios.

## Características

- **Chatbot Inteligente**: Interfaz tipo chat con respuestas instantáneas (HTMX) y búsqueda en tiempo real.
- **Panel de Admin**: ABM (Alta, Baja, Modificación) de mesas de examen.
- **Autenticación**: Login seguro para administradores.
- **Dockerizado**: Listo para desplegar con Docker y Docker Compose.
- **Base de Datos**: SQLite (ligera y contenida en el proyecto).

## Requisitos

- [Docker](https://www.docker.com/) y Docker Compose.
- Opcional: Go 1.24+ si quieres correrlo nativamente.

## 🚀 Ejecución Rápida (Recomendado)

La forma más fácil de correr el proyecto es con Docker.

1. **Construir y levantar el contenedor:**
   ```bash
   docker-compose up --build
   ```

2. **Acceder a la aplicación:**
   - **Chat Alumnos**: [http://localhost:8080](http://localhost:8080)
   - **Panel Admin**: [http://localhost:8080/admin](http://localhost:8080/admin)

3. **Credenciales por defecto:**
   - **Email**: `admin@unne.edu.ar`
   - **Password**: `admin123`
   *(Puedes cambiarlas en el archivo `docker-compose.yml`)*

## 🛠 Ejecución Local (Desarrollo)

Si prefieres correrlo sin Docker, necesitas tener **GCC** instalado (para SQLite).

1. **Instalar dependencias:**
   ```bash
   go mod tidy
   ```

2. **Configurar variables de entorno (Linux/Mac):**
   ```bash
   export ADMIN_EMAIL=admin@unne.edu.ar
   export ADMIN_PASSWORD=admin123
   ```

3. **Ejecutar:**
   ```bash
   go run cmd/server/main.go
   ```

## Estructura del Proyecto

El proyecto sigue una **Arquitectura Limpia (Clean Architecture)**:

```text
.
├── cmd/
│   └── server/       # Punto de entrada (Main)
├── internal/
│   ├── database/     # Conexión a SQLite
│   ├── handlers/     # Controladores HTTP (Gin)
│   ├── models/       # Estructuras de datos
│   └── repository/   # Consultas SQL
├── templates/        # Vistas HTML (Frontend)
├── Dockerfile        # Configuración de imagen Docker
└── docker-compose.yml
```

## Tecnologías

- **Backend**: Go + Gin Web Framework.
- **Frontend**: HTML5 + Bootstrap 5 + **HTMX** (para interactividad sin JS complejo).
- **Base de Datos**: SQLite3.
