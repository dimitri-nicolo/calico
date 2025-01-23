package command

func AuthenticateGCR(gcpCreds string) error {
	if _, err := Run("gcloud", []string{"auth", "activate-service-account", "--key-file", gcpCreds}); err != nil {
		return err
	}
	return nil
}
