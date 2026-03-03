package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/security"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage the BlackCat secrets vault",
}

var vaultSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Set a secret in the vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultSet,
}

var vaultGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a secret from the vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultGet,
}

var vaultListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secret keys in the vault",
	RunE:  runVaultList,
}

var vaultDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a secret from the vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultDelete,
}

func init() {
	rootCmd.AddCommand(vaultCmd)
	vaultCmd.AddCommand(vaultSetCmd, vaultGetCmd, vaultListCmd, vaultDeleteCmd)
	vaultCmd.PersistentFlags().String("vault-path", "", "Path to vault file (default: ~/.blackcat/vault.json)")
	vaultCmd.PersistentFlags().String("passphrase", "", "Vault passphrase (or set BLACKCAT_VAULT_PASSPHRASE)")
	vaultSetCmd.Flags().String("value", "", "Secret value (if not set, reads from stdin)")
}

func openVault(cmd *cobra.Command) (*security.Vault, error) {
	vaultPath, _ := cmd.Flags().GetString("vault-path")
	if vaultPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		vaultPath = filepath.Join(home, ".blackcat", "vault.json")
	}

	passphrase, _ := cmd.Flags().GetString("passphrase")
	if passphrase == "" {
		// Try environment variable
		passphrase = os.Getenv("BLACKCAT_VAULT_PASSPHRASE")
	}
	if passphrase == "" {
		// Prompt from stdin
		fmt.Fprint(os.Stderr, "Vault passphrase: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			passphrase = scanner.Text()
		}
		if scanner.Err() != nil {
			return nil, fmt.Errorf("failed to read passphrase: %w", scanner.Err())
		}
	}

	if passphrase == "" {
		return nil, fmt.Errorf("vault passphrase required")
	}

	vault, err := security.NewVault(vaultPath, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to open vault: %w", err)
	}
	return vault, nil
}

func runVaultSet(cmd *cobra.Command, args []string) error {
	key := args[0]

	vault, err := openVault(cmd)
	if err != nil {
		return err
	}

	value, _ := cmd.Flags().GetString("value")
	if value == "" {
		// Read from stdin
		fmt.Fprint(os.Stderr, "Value: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			value = scanner.Text()
		}
		if scanner.Err() != nil {
			return fmt.Errorf("failed to read value: %w", scanner.Err())
		}
	}

	if err := vault.Set(key, value); err != nil {
		return fmt.Errorf("failed to set secret: %w", err)
	}

	fmt.Printf("Secret '%s' saved\n", key)
	return nil
}

func runVaultGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	vault, err := openVault(cmd)
	if err != nil {
		return err
	}

	value, err := vault.Get(key)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	fmt.Println(value)
	return nil
}

func runVaultList(cmd *cobra.Command, args []string) error {
	vault, err := openVault(cmd)
	if err != nil {
		return err
	}

	keys := vault.List()
	if len(keys) == 0 {
		fmt.Println("no secrets in vault")
		return nil
	}

	for _, key := range keys {
		fmt.Println(key)
	}
	return nil
}

func runVaultDelete(cmd *cobra.Command, args []string) error {
	key := args[0]

	vault, err := openVault(cmd)
	if err != nil {
		return err
	}

	if err := vault.Delete(key); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	fmt.Printf("Secret '%s' deleted\n", key)
	return nil
}
