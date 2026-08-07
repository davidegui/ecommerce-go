# Sistema de Gestion de E-Commerce (Go) — Autonomo 2

Aplicacion de consola en Go. Avance del proyecto, no la version final.
Los datos se guardan **en memoria**: al cerrar el programa se pierden.

## Como ejecutar

```bash
go build ./...
go run main.go
```

## Estructura

```
ecommerce/
    go.mod              nombre del modulo
    main.go             menus, listas de datos y logica de negocio
    models/
        entity.go       interfaz Entity (polimorfismo)
        product.go      struct Producto + control de inventario
        client.go       struct Cliente
        order.go        structs Pedido e ItemPedido
    utils/
        errors.go       errores del sistema centralizados
```

## Funciones por modulo (segun el Autonomo 1)

**Productos:** registrarProducto, modificarProducto, eliminarProducto,
consultarDisponibles, buscarPorNombre

**Clientes:** registrarCliente, actualizarCliente, consultarClientes,
eliminarCliente

**Pedidos:** crearPedido, agregarProductoAPedido, calcularTotal (incluye el
descuento), confirmarPedido, listarPedidos

**Inventario:** consultarStock, actualizarStock, reponerStock, verAlertas, y
DescontarStock como metodo del producto

## Donde esta cada requisito de la Unidad 3

| Requisito | Donde |
|---|---|
| Encapsulacion | `models/`: campos en minuscula, constructores que validan, getters y setters |
| Setters con validacion | `CambiarNombre`, `CambiarPrecio`, `CambiarStock`, `CambiarEmail` |
| Manejo de errores | `utils/errors.go` con `errors.New`, verbo `%w` para agregar contexto, `errors.Is` en `mostrarError` |
| Interfaces | `models.Entity` con los metodos `ID()` y `Describir()` |
| Polimorfismo | Opcion 5 del menu: productos, clientes y pedidos en una misma lista `[]models.Entity` recorrida con un solo bucle |
| Funcion medianamente compleja | `calcularTotal` en `main.go`: valida stock de cada linea, suma el subtotal y aplica descuento escalonado |
| Structs, slices | structs en `models/`, slices para las listas de datos y el detalle del pedido |

## Descuentos implementados

| Subtotal | Descuento |
|---|---|
| menos de $100 | 0% |
| $100 a $299.99 | 5% |
| $300 a $699.99 | 10% |
| $700 o mas | 15% |

## Pendiente para la entrega final

Persistencia en archivos JSON, capa de servicios separada de la interfaz de
consola, reportes de ventas y mas validaciones en los menus.
