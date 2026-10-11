import { Modal, Pressable, View } from 'react-native';
import { fr } from '../i18n';
import { useStyles } from '../theme';
import { Button } from './Button';
import { Text } from './Text';

interface ConfirmDialogProps {
  visible: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel?: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  visible,
  title,
  message,
  confirmLabel,
  cancelLabel = fr.common.cancel,
  destructive = false,
  loading = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const styles = useStyles((t) => ({
    backdrop: {
      flex: 1,
      backgroundColor: t.colors.overlay,
      alignItems: 'center',
      justifyContent: 'center',
      padding: t.spacing.xl,
    },
    dialog: {
      width: '100%',
      maxWidth: 380,
      backgroundColor: t.colors.surface,
      borderRadius: t.radius.xl,
      padding: t.spacing.xl,
      gap: t.spacing.md,
    },
    actions: { gap: t.spacing.sm, marginTop: t.spacing.sm },
  }));
  return (
    <Modal
      transparent
      visible={visible}
      animationType="fade"
      onRequestClose={onCancel}
      statusBarTranslucent
    >
      <View style={styles.backdrop}>
        <Pressable
          style={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 }}
          accessibilityRole="button"
          accessibilityLabel={cancelLabel}
          onPress={onCancel}
        />
        <View style={styles.dialog} accessibilityViewIsModal>
          <Text variant="heading">{title}</Text>
          <Text color="textMuted">{message}</Text>
          <View style={styles.actions}>
            <Button
              label={confirmLabel}
              variant={destructive ? 'danger' : 'primary'}
              onPress={onConfirm}
              loading={loading}
            />
            <Button label={cancelLabel} variant="ghost" onPress={onCancel} disabled={loading} />
          </View>
        </View>
      </View>
    </Modal>
  );
}
