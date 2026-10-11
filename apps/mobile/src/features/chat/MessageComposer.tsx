import { useState } from 'react';
import { TextInput, View } from 'react-native';
import { fr } from '../../i18n';
import { MESSAGE_MAX } from '../../lib/validation';
import { TOUCH_TARGET, useStyles, useTheme } from '../../theme';
import { IconButton, icons } from '../../ui';

export function MessageComposer({ onSend }: { onSend: (body: string) => void }) {
  const { colors } = useTheme();
  const [text, setText] = useState('');
  const styles = useStyles((t) => ({
    bar: {
      flexDirection: 'row',
      alignItems: 'flex-end',
      gap: t.spacing.sm,
      paddingVertical: t.spacing.sm,
      borderTopWidth: 1,
      borderTopColor: t.colors.border,
      backgroundColor: t.colors.bg,
    },
    input: {
      flex: 1,
      minHeight: TOUCH_TARGET,
      maxHeight: 120,
      borderRadius: 22,
      borderWidth: 1.5,
      borderColor: t.colors.border,
      backgroundColor: t.colors.surface,
      paddingHorizontal: t.spacing.lg,
      paddingTop: 12,
      paddingBottom: 12,
      fontSize: t.fontSizes.body,
      color: t.colors.text,
      outlineWidth: 0,
    },
  }));
  const canSend = text.trim().length > 0;
  const send = () => {
    const body = text.trim();
    if (!body) return;
    setText('');
    onSend(body);
  };
  return (
    <View style={styles.bar}>
      <TextInput
        value={text}
        onChangeText={setText}
        placeholder={fr.chat.placeholder}
        placeholderTextColor={colors.textMuted}
        accessibilityLabel={fr.chat.inputLabel}
        multiline
        maxLength={MESSAGE_MAX}
        style={styles.input}
        selectionColor={colors.primary}
        testID="chat-input"
      />
      <IconButton
        glyph={icons.up}
        accessibilityLabel={fr.chat.send}
        tone="primary"
        disabled={!canSend}
        onPress={send}
        testID="chat-send"
      />
    </View>
  );
}
