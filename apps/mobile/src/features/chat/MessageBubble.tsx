import { Pressable, View } from 'react-native';
import { fr } from '../../i18n';
import { formatTime } from '../../lib/date';
import { useStyles } from '../../theme';
import { Text } from '../../ui';

interface MessageBubbleProps {
  body: string;
  createdAt: string;
  mine: boolean;
  /** Delivery line under my last message: "Lu" / "Envoyé" / "Échec". */
  status?: 'read' | 'sent' | 'sending' | 'failed';
  onRetry?: () => void;
}

export function MessageBubble({ body, createdAt, mine, status, onRetry }: MessageBubbleProps) {
  const styles = useStyles((t) => ({
    row: { paddingVertical: 2, flexDirection: 'row' },
    rowMine: { justifyContent: 'flex-end' },
    bubble: {
      maxWidth: '82%',
      borderRadius: t.radius.lg,
      paddingHorizontal: t.spacing.md,
      paddingVertical: t.spacing.sm,
      gap: 2,
    },
    mine: { backgroundColor: t.colors.primary, borderBottomRightRadius: 6 },
    theirs: {
      backgroundColor: t.colors.surface,
      borderWidth: 1,
      borderColor: t.colors.border,
      borderBottomLeftRadius: 6,
    },
    failed: { opacity: 0.7 },
    meta: { flexDirection: 'row', justifyContent: 'flex-end', gap: t.spacing.sm },
    statusRow: { alignItems: 'flex-end', paddingRight: 4, paddingBottom: 2 },
  }));
  const textColor = mine ? 'textOnPrimary' : 'text';
  const time = status === 'sending' ? fr.chat.sending : formatTime(createdAt);
  const label = fr.chat.messageLabel(mine, body, time);
  const content = (
    <View
      style={[
        styles.bubble,
        mine ? styles.mine : styles.theirs,
        status === 'failed' && styles.failed,
      ]}
    >
      <Text color={textColor} accessible={false}>
        {body}
      </Text>
      <View style={styles.meta}>
        <Text
          variant="caption"
          color={textColor}
          style={{ opacity: 0.75, fontSize: 11 }}
          accessible={false}
        >
          {time}
        </Text>
      </View>
    </View>
  );
  return (
    <View>
      <View style={[styles.row, mine && styles.rowMine]}>
        {status === 'failed' && onRetry ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={fr.chat.retrySend}
            onPress={onRetry}
            style={{ maxWidth: '82%' }}
          >
            {content}
          </Pressable>
        ) : (
          <View accessible accessibilityLabel={label} style={{ maxWidth: '82%' }}>
            {content}
          </View>
        )}
      </View>
      {mine && status && status !== 'sending' ? (
        <View style={styles.statusRow}>
          <Text
            variant="caption"
            color={status === 'failed' ? 'danger' : status === 'read' ? 'primary' : 'textMuted'}
            weight={status === 'read' ? '700' : '400'}
          >
            {status === 'read' ? fr.chat.read : status === 'failed' ? fr.chat.failed : fr.chat.sent}
          </Text>
        </View>
      ) : null}
    </View>
  );
}
