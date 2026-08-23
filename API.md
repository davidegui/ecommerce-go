# Documentación de los servicios web

Servidor: `http://localhost:8080`
Formato de intercambio: **JSON**
Sin dependencias externas: todo con `net/http` de la librería estándar.

## Cómo iniciar el servidor

```bash
go run main.go
```

Elegir la **opción 6** del menú principal. El servidor queda escuchando hasta
que se presione `Ctrl+C`.

## Formato de respuesta

Todas las respuestas tienen la misma estructura:

```json
{
  "exito": true,
  "mensaje": "Producto registrado correctamente",
  "datos": { }
}
```

Cuando algo falla:

```json
{
  "exito": false,
  "error": "el precio debe ser mayor a cero"
}
```

## Códigos de estado usados

| Código | Significado | Cuándo se devuelve |
|---|---|---|
| 200 | OK | Consulta o modificación exitosa |
| 201 | Created | Se registró un producto, cliente o pedido |
| 400 | Bad Request | Datos inválidos (precio negativo, correo mal escrito) |
| 404 | Not Found | El producto, cliente o pedido no existe |
| 409 | Conflict | Existe pero no se puede operar (stock insuficiente) |
| 500 | Internal Server Error | Fallo al leer o escribir los archivos JSON |

---

# Los servicios

## PRODUCTOS

### 1. Registrar producto
```
POST /api/productos
```
Cuerpo:
```json
{
  "nombre": "Teclado mecanico",
  "precio": 45.50,
  "stock": 20,
  "stock_minimo": 5
}
```
Respuesta: `201 Created` con el producto creado y su código asignado.

### 2. Consultar productos
```
GET /api/productos
GET /api/productos?disponibles=true
GET /api/productos?nombre=teclado
```
Sin parámetros devuelve el catálogo completo. Con `disponibles=true` solo los
que tienen stock. Con `nombre=` busca por coincidencia parcial, sin distinguir
mayúsculas.

### 3. Modificar producto
```
PUT /api/productos/{id}
```
Cuerpo (los campos omitidos no se modifican):
```json
{
  "precio": 49.99
}
```

### 4. Eliminar producto
```
DELETE /api/productos/P001
```

---

## CLIENTES

### 5. Registrar cliente
```
POST /api/clientes
```
Cuerpo:
```json
{
  "nombre": "Maria Lopez",
  "correo": "maria@correo.com",
  "telefono": "0991234567"
}
```
El correo se valida antes de registrar. Si no tiene formato válido devuelve 400.

### 6. Consultar clientes
```
GET /api/clientes
```

### 7. Actualizar cliente
```
PUT /api/clientes/{id}
```
Cuerpo (los campos omitidos no se modifican):
```json
{
  "telefono": "0987654321"
}
```

---

## PEDIDOS

### 8. Crear pedido
```
POST /api/pedidos
```
Cuerpo:
```json
{
  "cliente_id": "C001"
}
```
El cliente debe existir. El pedido nace vacío y en estado PENDIENTE.

### 9. Consultar pedidos
```
GET /api/pedidos
```

### 10. Agregar producto al pedido
```
POST /api/pedidos/{id}/productos
```
Cuerpo:
```json
{
  "producto_id": "P002",
  "cantidad": 3
}
```
Verifica que haya stock suficiente, pero **no descuenta inventario**: eso ocurre
al confirmar. Si no alcanza devuelve 409. El total se recalcula automáticamente.

### 11. Confirmar pedido
```
POST /api/pedidos/{id}/confirmar
```
Recalcula el total, descuenta el inventario de cada producto vendido, cierra el
pedido y devuelve las alertas de stock bajo generadas por la venta.

---

## INVENTARIO

### 12. Consultar inventario
```
GET /api/inventario
GET /api/inventario?producto=P001
```
Sin parámetros devuelve los productos con stock bajo. Con `producto=` devuelve
el estado de inventario de ese producto.

### 13. Actualizar inventario
```
PUT /api/inventario
```
Cuerpo:
```json
{
  "producto_id": "P003",
  "cantidad": 50,
  "operacion": "reponer"
}
```
La operación puede ser `"reponer"` (suma unidades al stock actual) o `"fijar"`
(establece un valor exacto).

---

## GENERAL

### 14. Resumen del sistema
```
GET /api/resumen
```
Devuelve la descripción de todos los registros —productos, clientes y pedidos—
recorriendo una única lista de tipo `models.Entity` con un solo bucle. Es la
demostración de **polimorfismo** del proyecto.

---

# Ejemplos de prueba

Con `curl` desde la terminal:

```bash
# Registrar un cliente
curl -X POST http://localhost:8080/api/clientes \
  -H "Content-Type: application/json" \
  -d '{"nombre":"Maria Lopez","correo":"maria@correo.com","telefono":"0991234567"}'

# Registrar un producto
curl -X POST http://localhost:8080/api/productos \
  -H "Content-Type: application/json" \
  -d '{"nombre":"Monitor 24","precio":189.99,"stock":8,"stock_minimo":2}'

# Crear un pedido y agregarle productos
curl -X POST http://localhost:8080/api/pedidos \
  -H "Content-Type: application/json" -d '{"cliente_id":"C001"}'

curl -X POST http://localhost:8080/api/pedidos/O001/productos \
  -H "Content-Type: application/json" -d '{"producto_id":"P001","cantidad":3}'

# Confirmar
curl -X POST http://localhost:8080/api/pedidos/O001/confirmar

# Ver el resumen del sistema
curl http://localhost:8080/api/resumen
```

En **Postman**: elegir el método, pegar la dirección, y en la pestaña Body
seleccionar `raw` y `JSON` para el cuerpo de las peticiones POST y PUT.
