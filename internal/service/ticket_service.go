// Business logic untuk tiket: generate ticket_number (random, BUKAN sequential
// — lihat catatan keamanan di LLD), hitung sla_due_at dari priority, validasi
// transisi status, dll. Handler memanggil service, service memanggil repository.
package service
