package engineapi

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/longhorn/backupstore"

	btypes "github.com/longhorn/backupstore/types"
	lhexec "github.com/longhorn/go-common-libs/exec"
	lhtypes "github.com/longhorn/go-common-libs/types"
	etypes "github.com/longhorn/longhorn-engine/pkg/types"

	"github.com/longhorn/longhorn-manager/datastore"
	"github.com/longhorn/longhorn-manager/types"
	"github.com/longhorn/longhorn-manager/util"

	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
)

const (
	backupStateInProgress = "in_progress"
	backupStateComplete   = "complete"
	backupStateError      = "error"
)

type BackupTargetClient struct {
	Image          string
	URL            string
	Credential     map[string]string
	ExecuteTimeout time.Duration
}

// NewBackupTargetClient returns the backup target client
func NewBackupTargetClient(engineImage, url string, credential map[string]string, executeTimeout time.Duration) *BackupTargetClient {
	return &BackupTargetClient{
		Image:          engineImage,
		URL:            url,
		Credential:     credential,
		ExecuteTimeout: executeTimeout,
	}
}

func NewBackupTargetClientFromBackupTarget(backupTarget *longhorn.BackupTarget, ds *datastore.DataStore) (*BackupTargetClient, error) {
	defaultEngineImage, err := ds.GetSettingValueExisted(types.SettingNameDefaultEngineImage)
	if err != nil {
		return nil, err
	}

	backupType, err := util.CheckBackupType(backupTarget.Spec.BackupTargetURL)
	if err != nil {
		return nil, err
	}

	var credential map[string]string
	if types.BackupStoreRequireCredential(backupType) {
		if backupTarget.Spec.CredentialSecret == "" {
			return nil, errors.Errorf("cannot access %s without credential secret", backupType)
		}

		credential, err = ds.GetCredentialFromSecret(backupTarget.Spec.CredentialSecret)
		if err != nil {
			return nil, err
		}
	}

	executeTimeout, err := ds.GetSettingAsInt(types.SettingNameBackupExecutionTimeout)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(executeTimeout) * time.Minute

	return NewBackupTargetClient(defaultEngineImage, backupTarget.Spec.BackupTargetURL, credential, timeout), nil
}

func (btc *BackupTargetClient) LonghornEngineBinary() string {
	return filepath.Join(types.GetEngineBinaryDirectoryOnHostForImage(btc.Image), "longhorn")
}

// getBackupCredentialEnv returns the environment variables as KEY=VALUE in string slice
func getBackupCredentialEnv(backupTarget string, credential map[string]string) ([]string, error) {
	envs := []string{}
	backupType, err := util.CheckBackupType(backupTarget)
	if err != nil {
		return envs, err
	}

	if !types.BackupStoreRequireCredential(backupType) || credential == nil {
		return envs, nil
	}

	switch backupType {
	case types.BackupStoreTypeS3:
		var missingKeys []string
		if credential[types.AWSAccessKey] == "" {
			missingKeys = append(missingKeys, types.AWSAccessKey)
		}
		if credential[types.AWSSecretKey] == "" {
			missingKeys = append(missingKeys, types.AWSSecretKey)
		}
		// If AWS IAM Role not present, then the AWS credentials must be exists
		if credential[types.AWSIAMRoleArn] == "" && len(missingKeys) > 0 {
			return nil, fmt.Errorf("could not backup to %s, missing %v in the secret", backupType, missingKeys)
		}
		if len(missingKeys) == 0 {
			envs = append(envs, fmt.Sprintf("%s=%s", types.AWSAccessKey, credential[types.AWSAccessKey]))
			envs = append(envs, fmt.Sprintf("%s=%s", types.AWSSecretKey, credential[types.AWSSecretKey]))
		}
		envs = append(envs, fmt.Sprintf("%s=%s", types.AWSEndPoint, credential[types.AWSEndPoint]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.AWSCert, credential[types.AWSCert]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.HTTPSProxy, credential[types.HTTPSProxy]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.HTTPProxy, credential[types.HTTPProxy]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.NOProxy, credential[types.NOProxy]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.VirtualHostedStyle, credential[types.VirtualHostedStyle]))
	case types.BackupStoreTypeCIFS:
		envs = append(envs, fmt.Sprintf("%s=%s", types.CIFSUsername, credential[types.CIFSUsername]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.CIFSPassword, credential[types.CIFSPassword]))
	case types.BackupStoreTypeAZBlob:
		envs = append(envs, fmt.Sprintf("%s=%s", types.AZBlobAccountName, credential[types.AZBlobAccountName]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.AZBlobAccountKey, credential[types.AZBlobAccountKey]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.AZBlobEndpoint, credential[types.AZBlobEndpoint]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.AZBlobCert, credential[types.AZBlobCert]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.HTTPSProxy, credential[types.HTTPSProxy]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.HTTPProxy, credential[types.HTTPProxy]))
		envs = append(envs, fmt.Sprintf("%s=%s", types.NOProxy, credential[types.NOProxy]))
	}
	return envs, nil
}

func (btc *BackupTargetClient) ExecuteEngineBinary(args ...string) (string, error) {
	envs, err := getBackupCredentialEnv(btc.URL, btc.Credential)
	if err != nil {
		return "", err
	}
	return lhexec.NewExecutor().Execute(envs, btc.LonghornEngineBinary(), args, btc.ExecuteTimeout)
}

func (btc *BackupTargetClient) ExecuteEngineBinaryWithTimeout(timeout time.Duration, args ...string) (string, error) {
	envs, err := getBackupCredentialEnv(btc.URL, btc.Credential)
	if err != nil {
		return "", err
	}
	return lhexec.NewExecutor().Execute(envs, btc.LonghornEngineBinary(), args, timeout)
}

func (btc *BackupTargetClient) ExecuteEngineBinaryWithoutTimeout(args ...string) (string, error) {
	envs, err := getBackupCredentialEnv(btc.URL, btc.Credential)
	if err != nil {
		return "", err
	}
	return lhexec.NewExecutor().Execute(envs, btc.LonghornEngineBinary(), args, lhtypes.ExecuteNoTimeout)
}

// parseBackupVolumeNamesList parses a list of backup volume names into a sorted string slice
func parseBackupVolumeNamesList(output string) ([]string, error) {
	data := map[string]struct{}{}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return nil, errors.Wrapf(err, "error parsing backup volume names: \n%s", output)
	}

	volumeNames := []string{}
	for volumeName := range data {
		if util.ValidateName(volumeName) {
			volumeNames = append(volumeNames, volumeName)
		}
	}
	sort.Strings(volumeNames)
	return volumeNames, nil
}

// BackupVolumeNameList returns a list of backup volume names
func (btc *BackupTargetClient) BackupVolumeNameList() ([]string, error) {
	output, err := btc.ExecuteEngineBinary("backup", "ls", "--volume-only", btc.URL)
	if err != nil {
		if types.ErrorIsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error listing backup volume names")
	}
	return parseBackupVolumeNamesList(output)
}

// parseBackupNamesList parses a list of backup names into a sorted string slice
func parseBackupNamesList(output, volumeName string) ([]string, error) {
	data := map[string]*BackupVolume{}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return nil, errors.Wrapf(err, "error parsing backup names: \n%s", output)
	}

	volumeData, ok := data[volumeName]
	if !ok {
		return nil, fmt.Errorf("cannot find the volume name %s in the data", volumeName)
	}

	backupNames := []string{}
	if volumeData.Messages[string(btypes.MessageTypeError)] != "" {
		return backupNames, errors.New(volumeData.Messages[string(btypes.MessageTypeError)])
	}
	for backupName := range volumeData.Backups {
		backupNames = append(backupNames, backupName)
	}
	sort.Strings(backupNames)
	return backupNames, nil
}

// BackupNameList returns a list of backup names
func (btc *BackupTargetClient) BackupNameList(destURL, volumeName string, credential map[string]string) ([]string, error) {
	if volumeName == "" {
		return nil, nil
	}
	output, err := btc.ExecuteEngineBinary("backup", "ls", "--volume", volumeName, btc.URL)
	if err != nil {
		if types.ErrorIsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error listing volume %s backup", volumeName)
	}
	return parseBackupNamesList(output, volumeName)
}

// BackupVolumeDelete deletes the backup volume from the remote backup target
func (btc *BackupTargetClient) BackupVolumeDelete(destURL, volumeName string, credential map[string]string) error {
	_, err := btc.ExecuteEngineBinaryWithoutTimeout("backup", "rm", "--volume", volumeName, btc.URL)
	if err != nil {
		if types.ErrorIsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "error deleting backup volume")
	}
	logrus.Infof("Complete deleting backup volume %s", volumeName)
	return nil
}

// parseBackupVolumeConfig parses a backup volume config
func parseBackupVolumeConfig(output string) (*BackupVolume, error) {
	backupVolume := new(BackupVolume)
	if err := json.Unmarshal([]byte(output), backupVolume); err != nil {
		return nil, errors.Wrapf(err, "error parsing one backup volume config: \n%s", output)
	}
	return backupVolume, nil
}

// BackupVolumeGet inspects a backup volume config with the given volume config URL
func (btc *BackupTargetClient) BackupVolumeGet(backupVolumeURL string, credential map[string]string) (*BackupVolume, error) {
	output, err := btc.ExecuteEngineBinary("backup", "inspect-volume", backupVolumeURL)
	if err != nil {
		if types.ErrorIsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error getting backup volume config %s", backupVolumeURL)
	}
	return parseBackupVolumeConfig(output)
}

// parseBackupConfig parses a backup config
func parseBackupConfig(output string) (*Backup, error) {
	backup := new(Backup)
	if err := json.Unmarshal([]byte(output), backup); err != nil {
		return nil, errors.Wrapf(err, "error parsing one backup config: \n%s", output)
	}
	return backup, nil
}

// BackupGet inspects a backup config with the given backup config URL
func (btc *BackupTargetClient) BackupGet(backupConfigURL string, credential map[string]string) (*Backup, error) {
	output, err := btc.ExecuteEngineBinary("backup", "inspect", backupConfigURL)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting backup config %s", backupConfigURL)
	}
	return parseBackupConfig(output)
}

// parseConfigMetadata parses the config metadata
func parseConfigMetadata(output string) (*ConfigMetadata, error) {
	metadata := new(ConfigMetadata)
	if err := json.Unmarshal([]byte(output), metadata); err != nil {
		return nil, errors.Wrapf(err, "error parsing config metadata: \n%s", output)
	}
	return metadata, nil
}

// BackupConfigMetaGet returns the config metadata with the given URL
func (btc *BackupTargetClient) BackupConfigMetaGet(url string, credential map[string]string) (*ConfigMetadata, error) {
	output, err := btc.ExecuteEngineBinary("backup", "head", url)
	if err != nil {
		if types.ErrorIsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error getting config metadata %s", url)
	}
	return parseConfigMetadata(output)
}

// BackupDelete deletes the backup from the remote backup target
func (btc *BackupTargetClient) BackupDelete(backupURL string, credential map[string]string) error {
	logrus.Infof("Start deleting backup %s", backupURL)
	_, err := btc.ExecuteEngineBinaryWithoutTimeout("backup", "rm", backupURL)
	if err != nil {
		if types.ErrorIsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "error deleting backup %v", backupURL)
	}
	logrus.Infof("Complete deleting backup %s", backupURL)
	return nil
}

// BackupCleanUpAllMounts clean up all mount points of backup store on the node
func (btc *BackupTargetClient) BackupCleanUpAllMounts() (err error) {
	_, err = btc.ExecuteEngineBinary("backup", "cleanup-all-mounts")
	if err != nil {
		return errors.Wrapf(err, "error clean up all mount points")
	}
	return nil
}

// SnapshotBackup calls engine binary
// TODO: Deprecated, replaced by gRPC proxy
func (e *EngineBinary) SnapshotBackup(engine *longhorn.Engine, snapName, backupName, backupTarget,
	backingImageName, backingImageChecksum, compressionMethod string, concurrentLimit int, storageClassName string,
	labels, credential, parameters map[string]string) (string, string, error) {
	if snapName == etypes.VolumeHeadName {
		return "", "", fmt.Errorf("invalid operation: cannot backup %v", etypes.VolumeHeadName)
	}
	// TODO: update when replacing this function
	snap, err := e.SnapshotGet(nil, snapName)
	if err != nil {
		return "", "", errors.Wrapf(err, "error getting snapshot '%s', volume '%s'", snapName, e.volumeName)
	}
	if snap == nil {
		return "", "", errors.Errorf("could not find snapshot '%s' to backup, volume '%s'", snapName, e.volumeName)
	}
	version, err := e.VersionGet(nil, true)
	if err != nil {
		return "", "", err
	}
	args := []string{"backup", "create", "--dest", backupTarget}
	if backingImageName != "" {
		args = append(args, "--backing-image-name", backingImageName)
		// TODO: Remove this if there is no backward compatibility
		if version.ClientVersion.CLIAPIVersion <= CLIVersionFour {
			args = append(args, "--backing-image-url", "deprecated-field")
		} else if backingImageChecksum != "" {
			args = append(args, "--backing-image-checksum", backingImageChecksum)
		}
	}
	if backupName != "" && version.ClientVersion.CLIAPIVersion > CLIVersionFour {
		args = append(args, "--backup-name", backupName)
	}
	for k, v := range labels {
		args = append(args, "--label", k+"="+v)
	}
	args = append(args, snapName)

	// get environment variables if backup for s3
	envs, err := getBackupCredentialEnv(backupTarget, credential)
	if err != nil {
		return "", "", err
	}
	output, err := e.ExecuteEngineBinaryWithoutTimeout(envs, args...)
	if err != nil {
		return "", "", err
	}
	backupCreateInfo := BackupCreateInfo{}
	if err := json.Unmarshal([]byte(output), &backupCreateInfo); err != nil {
		return "", "", err
	}

	logrus.Infof("Backup %v created for volume %v snapshot %v", backupCreateInfo.BackupID, e.Name(), snapName)
	return backupCreateInfo.BackupID, backupCreateInfo.ReplicaAddress, nil
}

// SnapshotBackupStatus calls engine binary
// TODO: Deprecated, replaced by gRPC proxy
func (e *EngineBinary) SnapshotBackupStatus(engine *longhorn.Engine, backupName, replicaAddress,
	replicaName string) (*longhorn.EngineBackupStatus, error) {
	args := []string{"backup", "status", backupName}
	if replicaAddress != "" {
		args = append(args, "--replica", replicaAddress)
	}

	// For now, we likely don't know the replica name here. Don't bother checking the binary version if we don't.
	if replicaName != "" {
		version, err := e.VersionGet(engine, true)
		if err != nil {
			return nil, err
		}
		if version.ClientVersion.CLIAPIVersion >= 9 {
			args = append(args, "--replica-instance-name", replicaName)
		}
	}

	output, err := e.ExecuteEngineBinary(args...)
	if err != nil {
		return nil, err
	}

	engineBackupStatus := &longhorn.EngineBackupStatus{}
	if err := json.Unmarshal([]byte(output), engineBackupStatus); err != nil {
		return nil, err
	}
	return engineBackupStatus, nil
}

// ConvertEngineBackupState converts longhorn engine backup state to Backup CR state
func ConvertEngineBackupState(state string) longhorn.BackupState {
	// https://github.com/longhorn/longhorn-engine/blob/9da3616/pkg/replica/backup.go#L20-L22
	switch state {
	case backupStateInProgress:
		return longhorn.BackupStateInProgress
	case backupStateComplete:
		return longhorn.BackupStateCompleted
	case backupStateError:
		return longhorn.BackupStateError
	default:
		return longhorn.BackupStateUnknown
	}
}

// BackupRestore calls engine binary
// TODO: Deprecated, replaced by gRPC proxy
func (e *EngineBinary) BackupRestore(engine *longhorn.Engine, backupTarget, backupName, backupVolumeName,
	lastRestored string, credential map[string]string, concurrentLimit int) error {
	backup := backupstore.EncodeBackupURL(backupName, backupVolumeName, backupTarget)

	// get environment variables if backup for s3
	envs, err := getBackupCredentialEnv(backupTarget, credential)
	if err != nil {
		return err
	}

	args := []string{"backup", "restore", backup}
	// TODO: Remove this compatible code and update the function signature
	//  when the manager doesn't support the engine v1.0.0 or older version.
	if lastRestored != "" {
		args = append(args, "--incrementally", "--last-restored", lastRestored)
	}

	if output, err := e.ExecuteEngineBinaryWithoutTimeout(envs, args...); err != nil {
		var taskErr TaskError
		if jsonErr := json.Unmarshal([]byte(output), &taskErr); jsonErr != nil {
			logrus.Warnf("Cannot unmarshal the restore error, maybe it's not caused by the replica restore failure: %v", jsonErr)
			return err
		}
		return taskErr
	}

	logrus.Infof("Backup %v restored for volume %v", backup, e.Name())
	return nil
}

// BackupRestoreStatus calls engine binary
// TODO: Deprecated, replaced by gRPC proxy
func (e *EngineBinary) BackupRestoreStatus(*longhorn.Engine) (map[string]*longhorn.RestoreStatus, error) {
	args := []string{"backup", "restore-status"}
	output, err := e.ExecuteEngineBinary(args...)
	if err != nil {
		return nil, err
	}
	replicaStatusMap := make(map[string]*longhorn.RestoreStatus)
	if err := json.Unmarshal([]byte(output), &replicaStatusMap); err != nil {
		return nil, err
	}
	return replicaStatusMap, nil
}

// CleanupBackupMountPoints calls engine binary
// TODO: Deprecated, replaced by gRPC proxy, just to match the interface
func (e *EngineBinary) CleanupBackupMountPoints() error {
	return errors.New(ErrNotImplement)
}
