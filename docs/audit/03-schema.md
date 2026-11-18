# Audit 03 — Database Schema

**Source:** `migrations/000001_init.sql` through `000064_seed_sample_music.sql` (applied 000001–000013 verified on fresh DB)

---

## Key Tables (ID Types, PKs, FKs, Constraints, Indexes)

### `users` (Migration 000002)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `uuid_generate_v4()` |
| `email` | VARCHAR(255) | NOT NULL, UNIQUE |
| `username` | VARCHAR(50) | NOT NULL, UNIQUE |
| `display_name` | VARCHAR(100) | NOT NULL |
| `password_hash` | TEXT | NOT NULL |
| `avatar_url` | TEXT | |
| `role` | VARCHAR(30) | NOT NULL, DEFAULT `'user'` |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE |
| `email_verified_at` | TIMESTAMPTZ | |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**Indexes:** `idx_users_email` (email), `idx_users_username` (username), `idx_users_created_at` (created_at)

---

### `auth_sessions` (Migration 000002)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `uuid_generate_v4()` |
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `refresh_token_hash` | TEXT | NOT NULL, UNIQUE |
| `user_agent` | TEXT | |
| `ip_address` | TEXT | |
| `expires_at` | TIMESTAMPTZ | NOT NULL |
| `revoked_at` | TIMESTAMPTZ | |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**Indexes:** `idx_auth_sessions_user_id`, `idx_auth_sessions_refresh_token_hash`, `idx_auth_sessions_expires_at`

---

### `artists` (Migration 000003)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK (no default in 000003; 000013 adds `DEFAULT gen_random_uuid()`) |
| `name` | VARCHAR(200) | NOT NULL |
| `slug` | VARCHAR(220) | NOT NULL, UNIQUE |
| `bio` | TEXT | |
| `image_url` | TEXT | |
| `is_verified` | BOOLEAN | NOT NULL, DEFAULT FALSE |
| `monthly_listeners` | BIGINT | NOT NULL, DEFAULT 0 |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | |
| `avatar_media_id` | UUID | FK → `media(id)` ON DELETE SET NULL (added 000013) |
| `banner_media_id` | UUID | FK → `media(id)` ON DELETE SET NULL (added 000013) |
| `country` | VARCHAR(2) | (added 000013) |

**Indexes:** `idx_artists_name`, `idx_artists_monthly_listeners`, `idx_artists_slug` (000013), `idx_artists_avatar_media_id`, `idx_artists_banner_media_id`

---

### `albums` (Migration 000003)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK (no default in 000003; 000013 adds `DEFAULT gen_random_uuid()`) |
| `artist_id` | UUID | NOT NULL, FK → `artists(id)` ON DELETE CASCADE |
| `title` | VARCHAR(250) | NOT NULL |
| `slug` | VARCHAR(270) | NOT NULL |
| `cover_url` | TEXT | |
| `release_date` | DATE | |
| `album_type` | VARCHAR(50) | NOT NULL, DEFAULT `'album'` |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | |
| `cover_media_id` | UUID | FK → `media(id)` ON DELETE SET NULL (added 000013) |

**Constraints:** `albums_artist_id_slug_key` UNIQUE (`artist_id`, `slug`)  
**Indexes:** `idx_albums_artist_id`, `idx_albums_release_date`, `idx_albums_cover_media_id` (000013)

---

### `tracks` (Migration 000003)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK (no default in 000003; 000013 adds `DEFAULT gen_random_uuid()`) |
| `artist_id` | UUID | NOT NULL, FK → `artists(id)` ON DELETE CASCADE |
| `album_id` | UUID | FK → `albums(id)` ON DELETE SET NULL |
| `title` | VARCHAR(250) | NOT NULL |
| `slug` | VARCHAR(270) | NOT NULL |
| `duration_seconds` | INT | NOT NULL, DEFAULT 0 |
| `track_number` | INT | |
| `explicit` | BOOLEAN | NOT NULL, DEFAULT FALSE |
| `audio_url` | TEXT | |
| `cover_url` | TEXT | |
| `play_count` | BIGINT | NOT NULL, DEFAULT 0 |
| `is_public` | BOOLEAN | NOT NULL, DEFAULT TRUE |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | |
| `audio_media_id` | UUID | FK → `media(id)` ON DELETE SET NULL (added 000013) |
| `cover_media_id` | UUID | FK → `media(id)` ON DELETE SET NULL (added 000013) |

**Constraints:** `tracks_artist_id_slug_key` UNIQUE (`artist_id`, `slug`)  
**Indexes:** `idx_tracks_artist_id`, `idx_tracks_album_id`, `idx_tracks_is_public`, `idx_tracks_play_count`, `idx_tracks_audio_media_id`, `idx_tracks_cover_media_id` (000013)

---

### `genres` (Migration 000003)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK |
| `name` | VARCHAR(100) | NOT NULL, UNIQUE |
| `slug` | VARCHAR(120) | NOT NULL, UNIQUE |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**Indexes:** `idx_genres_name`

---

### `track_genres` (Migration 000003)
| Column | Type | Constraints |
|--------|------|-------------|
| `track_id` | UUID | NOT NULL, FK → `tracks(id)` ON DELETE CASCADE |
| `genre_id` | UUID | NOT NULL, FK → `genres(id)` ON DELETE CASCADE |

**PK:** (`track_id`, `genre_id`)  
**Indexes:** `idx_track_genres_track_id`, `idx_track_genres_genre_id`

---

### `playlists` (Migration 000004)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `name` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | NULL |
| `cover_url` | TEXT | NULL |
| `is_public` | BOOLEAN | NOT NULL, DEFAULT FALSE |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**Indexes:** `idx_playlists_user_id`, `idx_playlists_is_public`

---

### `playlist_tracks` (Migration 000004)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `playlist_id` | UUID | NOT NULL, FK → `playlists(id)` ON DELETE CASCADE |
| `track_id` | UUID | NOT NULL, FK → `tracks(id)` ON DELETE CASCADE |
| `position` | INT | NOT NULL, CHECK (`position > 0`) |
| `added_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**Constraints:** `uq_playlist_track` UNIQUE (`playlist_id`, `track_id`); `uq_playlist_position` UNIQUE (`playlist_id`, `position`)  
**Indexes:** `idx_playlist_tracks_playlist_id`, `idx_playlist_tracks_track_id`

---

### `queue_items` (Migration 000008)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `track_id` | UUID | NOT NULL, FK → `tracks(id)` ON DELETE CASCADE |
| `position` | INT | NOT NULL |
| `added_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

---

### `liked_tracks` (Migration 000005)
| Column | Type | Constraints |
|--------|------|-------------|
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `track_id` | UUID | NOT NULL, FK → `tracks(id)` ON DELETE CASCADE |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**PK:** (`user_id`, `track_id`)

---

### `liked_albums` (Migration 000005)
| Column | Type | Constraints |
|--------|------|-------------|
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `album_id` | UUID | NOT NULL, FK → `albums(id)` ON DELETE CASCADE |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**PK:** (`user_id`, `album_id`)

---

### `followed_artists` (Migration 000009)
| Column | Type | Constraints |
|--------|------|-------------|
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `artist_id` | UUID | NOT NULL, FK → `artists(id)` ON DELETE CASCADE |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**PK:** (`user_id`, `artist_id`)

---

### `play_history` / `listening_history` (Migration 000010)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `track_id` | UUID | NOT NULL, FK → `tracks(id)` ON DELETE CASCADE |
| `duration_seconds` | INT | |
| `completed` | BOOLEAN | |
| `played_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**Indexes:** `idx_play_history_user_id`, `idx_play_history_played_at`

---

### `notifications` (Migration 000007)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `type` | VARCHAR(50) | NOT NULL |
| `title` | VARCHAR(200) | NOT NULL |
| `message` | TEXT | |
| `data` | JSONB | |
| `read_at` | TIMESTAMPTZ | |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |

**Indexes:** `idx_notifications_user_id`, `idx_notifications_read_at`, `idx_notifications_created_at`

---

### `subscriptions` (Migration 000006)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `user_id` | UUID | NOT NULL, FK → `users(id)` ON DELETE CASCADE |
| `plan_id` | UUID | NOT NULL, FK → `subscription_plans(id)` |
| `status` | VARCHAR(30) | NOT NULL |
| `current_period_start` | TIMESTAMPTZ | NOT NULL |
| `current_period_end` | TIMESTAMPTZ | NOT NULL |
| `cancel_at_period_end` | BOOLEAN | NOT NULL, DEFAULT FALSE |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | |

---

### `payments` (Migration 000006)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `subscription_id` | UUID | FK → `subscriptions(id)` |
| `amount` | BIGINT | NOT NULL (cents/toman) |
| `currency` | VARCHAR(3) | NOT NULL |
| `status` | VARCHAR(30) | NOT NULL |
| `provider` | VARCHAR(30) | NOT NULL (zarinpal/idpay) |
| `provider_payment_id` | VARCHAR(100) | |
| `metadata` | JSONB | |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | |

---

### `lyrics` (Migration 000012)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `track_id` | UUID | NOT NULL, FK → `tracks(id)` ON DELETE CASCADE |
| `content` | TEXT | NOT NULL |
| `source` | VARCHAR(30) | NOT NULL |
| `synced` | BOOLEAN | NOT NULL, DEFAULT FALSE |
| `language` | VARCHAR(10) | DEFAULT `'fa'` |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | |

**Indexes:** `idx_lyrics_track_id`, `idx_lyrics_source`

---

### `media` (Migration 000013)
| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` |
| `kind` | VARCHAR(50) | NOT NULL |
| `url` | TEXT | NOT NULL |
| `mime_type` | VARCHAR(120) | |
| `size_bytes` | BIGINT | |
| `width` | INT | |
| `height` | INT | |
| `duration_seconds` | INT | |
| `bitrate` | INT | |
| `sample_rate` | INT | |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | |

**Indexes:** `idx_media_kind`, `idx_media_created_at`

---

## ID Type Consistency Check

| Endpoint Pattern | Swagger ID Type | Actual DB Column Type | Match? |
|------------------|-----------------|----------------------|--------|
| `/catalog/tracks/:trackID` | string (UUID) | `tracks.id` UUID | ✅ |
| `/catalog/albums/:albumID` | string (UUID) | `albums.id` UUID | ✅ |
| `/catalog/artists/:artistID` | string (UUID) | `artists.id` UUID | ✅ |
| `/library/tracks/:trackId` | **integer** (Swagger) | `tracks.id` UUID | ❌ **MISMATCH** |
| `/library/albums/:albumId` | **integer** (Swagger) | `albums.id` UUID | ❌ **MISMATCH** |
| `/library/artists/:artistId` | **integer** (Swagger) | `artists.id` UUID | ❌ **MISMATCH** |
| `/playlists/:id` | string (UUID) | `playlists.id` UUID | ✅ |
| `/playlists/:id/tracks/:trackId` | mixed | `playlist_tracks.track_id` UUID | ⚠️ param name `trackId` but UUID |
| `/history/:id` | string (UUID) | `play_history.id` UUID | ✅ |
| `/library/history` | uses `track_id` body | `play_history.track_id` UUID | ✅ (body) |
| `/playlist/:id/tracks/:trackId` | string | `playlist_tracks.track_id` UUID | ⚠️ |

**Critical Finding:** Library module Swagger docs show integer IDs (`:trackId` as `integer`, `:albumId` as `integer`, `:artistId` as `integer`) but all catalog entities use **UUID**. Handler uses `parseUUIDParam` which returns string, but Swagger is wrong.

---

## FK ON DELETE Behavior Summary

| Child Table | Parent Table | ON DELETE |
|-------------|--------------|-----------|
| `auth_sessions` | `users` | CASCADE |
| `artists` (avatar/banner) | `media` | SET NULL |
| `albums` | `artists` | CASCADE |
| `albums` (cover) | `media` | SET NULL |
| `tracks` | `artists` | CASCADE |
| `tracks` | `albums` | SET NULL |
| `tracks` (audio/cover) | `media` | SET NULL |
| `playlists` | `users` | CASCADE |
| `playlist_tracks` | `playlists` | CASCADE |
| `playlist_tracks` | `tracks` | CASCADE |
| `queue_items` | `users` / `tracks` | CASCADE |
| `liked_tracks` | `users` / `tracks` | CASCADE |
| `liked_albums` | `users` / `albums` | CASCADE |
| `followed_artists` | `users` / `artists` | CASCADE |
| `play_history` | `users` / `tracks` | CASCADE |
| `notifications` | `users` | CASCADE |
| `subscriptions` | `users` | CASCADE |
| `payments` | `subscriptions` | (no FK visible in migration) |

---

## Missing / Weak Constraints

| Table | Missing Constraint |
|-------|-------------------|
| `tracks` | No CHECK on `duration_seconds >= 0` |
| `playlist_tracks` | `position > 0` CHECK exists ✅ |
| `analytics` events | No table in migrations (logged via events/outbox) |
| `payments` | No FK to `subscriptions` in migration 000006 |
| `subscription_plans` | Not in provided migrations (referenced by `subscriptions.plan_id`) |
| `refresh_tokens` | `auth_sessions.refresh_token_hash` UNIQUE ✅ but no CHECK on `expires_at > NOW()` |

---

*Cite file:line from `migrations/000002_create_auth_tables.sql`, `000003_create_catalog_tables.sql`, `000004_create_playlist_tables.sql`, `000005_create_library_tables.sql`, `000006_create_subscription_tables.sql`, `000007_create_notifications.sql`, `000008_create_queue_items.sql`, `000009_create_follow_tables.sql`, `000010_create_listening_history.sql`, `000012_create_lyrics_tables.sql`, `000013_refactor_catalog_many_to_many.sql`.*