package normalize

// HashDrifted reports whether a server-computed content hash indicates the stored
// resource changed outside of the managing tool, by comparing a recorded baseline
// hash against the one currently returned by the API.
//
// It returns false (i.e. "no drift") whenever there is no trustworthy baseline to
// compare against: an empty recorded hash, or an empty remote hash. This mirrors the
// connected_app data_hash contract in the Terraform provider — drift detection is
// forward-looking and the first observation after adopting hashing establishes the
// baseline rather than being flagged as drift.
//
// It is used by both the Terraform provider's connected_app Read path and the
// Crossplane observe decorator so the two stay byte-for-byte consistent.
func HashDrifted(recordedHash, remoteHash string) bool {
	if recordedHash == "" || remoteHash == "" {
		return false
	}
	return recordedHash != remoteHash
}
