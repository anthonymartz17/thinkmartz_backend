package feed

// Repository defines the methods for reading post data from Postgres to
// rebuild a user's feed on a cache miss. Methods are added as hydration
// needs grow (e.g. ListPostsBefore).
type Repository interface {
}
