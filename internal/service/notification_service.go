// Kirim notifikasi email ke requester saat ticket dibuat & saat status berubah.
// v1: email only (lihat keputusan arsitektur — WA channel ditunda).
// Dipanggil secara async (goroutine) dari ticket_service supaya tidak blocking
// response API.
package service
