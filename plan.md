# Ansible Re-engineering Plan

> **⚠️ Catatan Penting:** Rata-rata kode di repo ini sudah berjalan dan telah terbukti bekerja di production.
> Jika perlu re-engineer atau menyembunyikan data yang terekspos, **lakukan perubahan seminor mungkin** —
> jangan refactor struktur task, jangan ganti nama variabel yang sudah jalan, cukup target yang spesifik saja.

---

## Hasil Scan — Temuan

### 🔴 Keamanan: Credentials Hardcoded (TRACKED BY GIT)

| File | Masalah |
|------|---------|
| `roles/databases/redis/defaults/main.yml` | `redis_password: "L1nkit360"`, `redis_username: "Linkit360"` — literal password |
| `roles/platforms/gitlab/defaults/main.yml` | `gitlab_root_password: "test123!!!"` — CONFLICT dengan `group_vars/platforms.yml` yang sudah punya nilai berbeda |
| `roles/storages/minio/defaults/main.yml` | `minio_root_password: "minioadmin123"` — literal password |
| `roles/messagings/debezium/templates/docker-compose.yml` | SASL passwords: `"dbz#2026!"`, `"usrAdm#46!"`, `"ui_secret_password"` hardcoded |
| `roles/messagings/debezium/templates/docker-compose-psql.yml` | `POSTGRES_PASSWORD: 'Pg2026#123!'` hardcoded |

> **Note:** File inventory (`baremetal.ini`, `production.ini`, `minutex-aws.ini`) yang berisi real IP + password sudah di-gitignore dengan benar ✅

### 🟠 Bug .gitignore: Template Files Tidak Ter-track

Pattern `.gitignore` saat ini:
```
**/docker-compose.yml
```
Pattern ini **juga mengecualikan** file template di dalam `roles/.../templates/docker-compose.yml` —
padahal template adalah source code yang harus di-track git. Ini bug yang perlu diperbaiki.

### 🟡 Role Structure: Tidak Konsisten

Beberapa role tidak punya folder standar (`defaults/`, `handlers/`, `templates/`):

| Role | defaults | handlers | templates |
|------|:---:|:---:|:---:|
| `apps/application` | ❌ | ❌ | ❌ |
| `base/common` | ❌ | ❌ | ❌ |
| `databases/elasticsearch` | ❌ | ✅ | ✅ |
| `databases/postgres` | ❌ | ❌ | ❌ |
| `iam/keycloak` | ✅ | ❌ | ✅ |
| `messagings/rabbitmq` | ❌ | ❌ | ❌ |
| `observability/grafana` | ❌ | ❌ | ❌ |
| `observability/node-exporter` | ❌ | ❌ | ❌ |
| `orchestrations/kind` | ✅ | ❌ | ❌ |
| `orchestrations/kubernetes` | ❌ | ❌ | ❌ |
| `storages/minio` | ✅ | ❌ | ✅ |
| `testings/k6` | ❌ | ❌ | ❌ |

### 🟡 Group_vars: Belum Semua Variabel Terpusat

Variabel yang masih hardcoded di `defaults/` padahal mestinya di `group_vars/`:

| group_vars file | Yang kurang |
|----------------|-------------|
| `databases.yml` | redis credentials, elasticsearch vars, postgres vars |
| `observability.yml` | grafana, prometheus, node-exporter vars |
| `orchestrations.yml` | temporal vars |
| *(belum ada)* `storages.yml` | minio credentials |
| *(belum ada)* `iam.yml` | keycloak config |
| *(belum ada)* `messagings.yml` | debezium SASL passwords, rabbitmq config |

---

## Rencana Implementasi

### Phase 1 — Fix .gitignore & Rename Templates

**Tujuan:** Agar file template di `templates/` folder ter-track oleh git.

**Langkah:**
1. Rename semua `docker-compose.yml` di dalam folder `templates/` → `docker-compose.yml.j2`
   - `roles/messagings/debezium/templates/docker-compose.yml` → `docker-compose.yml.j2`
   - `roles/messagings/debezium/templates/docker-compose-psql.yml` → `docker-compose-psql.yml.j2`
   - Cek semua role lain apakah ada template yang terdampak
2. Update task yang merujuk ke template tersebut (cari `src:` yang point ke `docker-compose.yml`)
3. Update `.gitignore`: ganti `**/docker-compose.yml` dengan pattern yang tidak menyentuh `templates/`

**Contoh fix .gitignore:**
```
# sebelum:
**/docker-compose.yml

# sesudah: hanya ignore file docker-compose yang di-generate/deploy, bukan di templates/
docker-compose.yml
!**/templates/*.j2
```

---

### Phase 2 — Sembunyikan Credentials Hardcoded

> **Prinsip:** Seminor mungkin. Cukup ganti nilai literal dengan variable reference. Jangan ubah nama var atau struktur.

**2a. Redis** — `roles/databases/redis/defaults/main.yml`
- Hapus nilai literal `"L1nkit360"` dan `"Linkit360"`
- Ganti dengan reference: `"{{ redis_password }}"` dan `"{{ redis_username }}"`
- Pindahkan nilai asli ke `inventory/group_vars/databases.yml`

**2b. GitLab** — `roles/platforms/gitlab/defaults/main.yml`
- Hapus `gitlab_root_password: "test123!!!"` dari defaults
- Nilai yang benar sudah ada di `group_vars/platforms.yml` → tidak perlu duplikasi

**2c. MinIO** — `roles/storages/minio/defaults/main.yml`
- Hapus `minio_root_password: "minioadmin123"` dari defaults
- Pindahkan ke `inventory/group_vars/storages.yml` (buat baru)

**2d. Debezium Templates** — `roles/messagings/debezium/templates/`
- Tambah 3 variabel di `roles/messagings/debezium/defaults/main.yml`:
  ```yaml
  debezium_kafka_debezium_password: "{{ debezium_kafka_debezium_password }}"
  debezium_kafka_admin_password: "{{ debezium_kafka_admin_password }}"
  debezium_kafka_ui_password: "{{ debezium_kafka_ui_password }}"
  ```
- Ganti hardcoded string di template dengan variable reference
- Pindahkan nilai asli ke `inventory/group_vars/messagings.yml` (buat baru)

---

### Phase 3 — Buat & Update group_vars

> **Prinsip:** Hanya tambah yang memang kurang. Jangan pindah atau rename variabel yang sudah jalan.

**Buat baru:**
- `inventory/group_vars/storages.yml` — minio credentials + config
- `inventory/group_vars/iam.yml` — keycloak config
- `inventory/group_vars/messagings.yml` — debezium SASL passwords, rabbitmq

**Update (append only):**
- `inventory/group_vars/databases.yml` — tambahkan redis credentials, elasticsearch vars
- `inventory/group_vars/observability.yml` — tambahkan grafana, prometheus, node-exporter
- `inventory/group_vars/orchestrations.yml` — tambahkan temporal vars yang masih di defaults

---

### Phase 4 — Standardisasi Struktur Role

> **Prinsip:** Cukup buat file/folder yang kurang. Jangan ubah task yang sudah ada.

Untuk 12 role yang belum lengkap, buat:
- `defaults/main.yml` — variabel default kosong atau minimal, dengan komentar
- `handlers/main.yml` — handler restart service yang sesuai role
- `templates/` folder — buat dengan `.gitkeep` jika memang tidak ada template

**Role yang perlu dibenahi:**
`apps/application`, `base/common`, `databases/elasticsearch` (cukup defaults), `databases/postgres`,
`iam/keycloak` (cukup handlers), `messagings/rabbitmq`, `observability/grafana`, `observability/node-exporter`,
`orchestrations/kind` (cukup handlers + templates), `orchestrations/kubernetes`, `storages/minio` (cukup handlers), `testings/k6`

---

### Phase 5 — Final Verification

1. Grep untuk memastikan tidak ada password/secret literal di file `.yml` atau `.j2` yang di-track git
2. Verify semua `notify:` di task punya handler yang cocok di `handlers/main.yml`
3. Pastikan tidak ada duplikasi definisi variabel antara `defaults/` dan `group_vars/`
4. Review `.gitignore` final — pastikan semua file sensitif ter-ignore, semua template ter-track

---

## Urutan Eksekusi di Agent Mode

```
1. Phase 1 → 2 → 3 → 4 → 5
2. Setiap phase selesai, lakukan quick verify sebelum lanjut
3. Jika ada konflik variabel, prioritaskan nilai di group_vars/ (lebih spesifik dari defaults/)
```
