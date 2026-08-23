# Sistema de Gestión para E-Commerce

Proyecto final de **Programación Orientada a Objetos** — desarrollado en Go.

> **Datos del grupo:** [completar con los nombres de los integrantes]
> **Fecha:** [completar]

---

## Objetivo del programa

Desarrollar un sistema de gestión para un comercio electrónico que permita
administrar productos, clientes, pedidos e inventario, aplicando los conceptos
de programación orientada a objetos que ofrece Go: encapsulación mediante campos
privados y métodos de acceso, polimorfismo mediante interfaces, manejo idiomático
de errores, persistencia mediante serialización JSON y manejo seguro de la
concurrencia.

El sistema puede usarse de dos formas independientes: como aplicación de consola
con menús interactivos, o como servidor web que expone sus funcionalidades a
través de servicios web que responden en formato JSON.

---

## Cómo ejecutar

Requiere Go 1.22 o superior.

```bash
go build ./...     # compila todos los paquetes
go run main.go     # ejecuta el programa
```

Al iniciar, el programa carga los datos guardados en la carpeta `data/`. Si los
archivos no existen se crean automáticamente.

- **Opciones 1 a 5:** aplicación de consola.
- **Opción 6:** inicia el servidor web en `http://localhost:8080`.

La documentación completa de los servicios web está en [API.md](API.md).

---

## Estructura del proyecto

```
ecommerce/
    go.mod                  nombre del módulo
    main.go                 menús de la aplicación de consola
    models/
        entity.go           interfaz Entity (polimorfismo)
        product.go          entidad Producto + control de inventario
        client.go           entidad Cliente
        order.go            entidades Pedido e ItemPedido
    services/
        service.go          lógica de negocio compartida
        reports.go          reportes concurrentes (goroutines y canales)
    storage/
        storage.go          persistencia en archivos JSON
    api/
        server.go           servicios web (net/http)
    utils/
        errors.go           errores del sistema centralizados
        money.go            redondeo de valores monetarios
    data/                   archivos JSON con los datos guardados
    *_test.go               pruebas unitarias (models/ y services/)
```

### Pruebas

```bash
go test ./...          # ejecuta las 60 pruebas
go test -race ./...    # incluye el detector de condiciones de carrera
go test -v ./...       # muestra el detalle de cada prueba
```

### Por qué está dividido así

La lógica de negocio vive en `services/` y **no se repite** entre la consola y
los servicios web: los dos llaman a las mismas funciones. Si mañana se agrega
una tercera interfaz (una aplicación de escritorio, por ejemplo), tampoco habría
que duplicar nada.

Los modelos no conocen ni la consola ni el servidor: solo saben validar y
proteger sus propios datos.

---

## Funcionalidades

### Gestión de productos
Registrar, modificar, eliminar, consultar disponibles y buscar por nombre con
coincidencia parcial.

### Gestión de clientes
Registrar con validación de correo, actualizar, consultar y eliminar.

### Gestión de pedidos
Crear pedidos, agregar productos verificando disponibilidad, calcular el total
con descuentos escalonados y confirmar descontando inventario.

### Gestión de inventario
Consultar existencias, fijar un valor exacto, reponer sumando unidades y generar
alertas automáticas cuando un producto llega a su stock mínimo.

### Reportes concurrentes
Tres reportes independientes —ventas del periodo, productos más vendidos y
estado del inventario— calculados en paralelo mediante goroutines, cuyos
resultados se recogen a través de un canal. Incluye además un monitor de stock
que envía las alertas por un canal a medida que las detecta.

### Servicios web
Dieciséis endpoints HTTP que exponen todas las funcionalidades anteriores y
responden en JSON.

---

## Función principal del sistema

`CalcularTotal` en `services/service.go` es la función de mayor complejidad del
proyecto. Está dividida en cuatro pasos comentados:

1. Verifica que el pedido tenga líneas.
2. Recorre el detalle, valida el stock disponible de cada producto y suma el
   subtotal usando el precio congelado en cada línea.
3. Aplica el descuento escalonado según el monto.
4. Calcula el descuento en dinero y lo resta del subtotal.

Si cualquier producto no tiene stock suficiente, la función rechaza el pedido
completo sin modificar nada.

### Reglas de descuento

| Subtotal | Descuento |
|---|---|
| menos de $100 | 0% |
| $100 a $299.99 | 5% |
| $300 a $699.99 | 10% |
| $700 o más | 15% |

Los tramos se evalúan de mayor a menor. En orden inverso, un pedido de $800
entraría en el primer tramo y recibiría 5% en lugar de 15%.

---

## Conceptos aplicados

| Concepto | Dónde está |
|---|---|
| **Encapsulación** | `models/`: campos privados, constructores que validan, getters y setters |
| **Setters con validación** | `CambiarNombre`, `CambiarPrecio`, `CambiarStock`, `CambiarEmail` |
| **Manejo de errores** | `utils/errors.go` con `errors.New`, contexto con `%w`, reconocimiento con `errors.Is` |
| **Interfaces** | `models.Entity` con los métodos `ID()` y `Describir()` |
| **Polimorfismo** | Los tres modelos cumplen `Entity`; una sola lista y un solo bucle los recorren |
| **Serialización JSON** | `MarshalJSON` y `UnmarshalJSON` en cada entidad, con struct puente |
| **Persistencia** | `storage/`: un archivo JSON por entidad en `data/` |
| **Servicios web** | `api/server.go`: 16 endpoints con `net/http` |
| **Concurrencia** | `services/service.go`: `sync.Mutex` protegiendo las listas; `services/reports.go`: goroutines, canales y `sync.WaitGroup` |
| **Testing** | 60 pruebas unitarias en `models/` y `services/`, ejecutables con `go test ./...` |
| **Structs, slices, maps** | Structs en `models/`, slices en las listas y en el detalle del pedido |

### Sobre la serialización

El paquete `encoding/json` solo puede acceder a campos exportados, es decir los
que empiezan en mayúscula. Como los campos de las entidades son privados, la
serialización automática devolvería objetos vacíos.

La solución no fue hacer públicos los campos —eso habría eliminado la
encapsulación— sino implementar `MarshalJSON` y `UnmarshalJSON` en cada entidad
usando un struct auxiliar privado que sirve de puente entre el objeto encapsulado
y el formato JSON.

`UnmarshalJSON` además vuelve a validar los datos leídos del archivo, porque un
archivo en disco puede haber sido editado a mano con valores inválidos.

### Sobre la concurrencia

El servidor web de Go atiende **cada petición en una goroutine distinta**. Eso
significa que varias operaciones pueden estar ejecutándose al mismo tiempo sobre
las mismas listas de datos.

Sin protección, dos peticiones simultáneas que registren productos leerían el
mismo estado de la lista, escribirían encima, y uno de los dos registros se
perdería sin ningún aviso. A ese problema se lo llama **condición de carrera**.

La solución fue proteger las listas con un candado (`sync.Mutex`) en
`services/service.go`. Cada función que lee o modifica los datos toma el candado
antes de empezar y lo libera con `defer` al terminar:

```go
func RegistrarProducto(...) (*models.Producto, error) {
    candado.Lock()
    defer candado.Unlock()
    ...
}
```

Se usa `defer` porque garantiza que el candado se libere aunque la función salga
antes de tiempo por un error. Un candado que no se libera deja el programa
esperando para siempre un turno que nunca llega.

**Detalle importante:** el mutex de Go no es reentrante. Si una función que ya
tomó el candado llama a otra que intenta tomarlo de nuevo, el programa se
bloquea esperándose a sí mismo. Por eso existen versiones internas como
`buscarProductoSinCandado`, que hacen el trabajo sin tomar el candado y son las
que usan las funciones que ya lo tienen.

#### Comprobación

Go incluye un detector de condiciones de carrera:

```bash
go build -race -o ecommerce .
```

Ejecutando el servidor compilado así y lanzando 60 peticiones simultáneas de
registro, se obtuvieron los 60 productos esperados y **cero condiciones de
carrera detectadas**.

---

## Estado del proyecto

Implementado y funcionando: los cuatro módulos completos, persistencia en JSON,
los dieciséis servicios web, reportes concurrentes con goroutines y canales,
acceso concurrente protegido, sesenta pruebas unitarias, validaciones en todas
las entidades y manejo de errores en toda la cadena.

Mejoras posibles para versiones futuras: autenticación de los servicios web,
reportes de ventas por periodo, categorías de producto e historial de cambios de
precio.
