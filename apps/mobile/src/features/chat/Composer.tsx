import React, { useState } from "react";
import { Pressable, StyleSheet, TextInput, View } from "react-native";
import { MESSAGE_MAX } from "../../lib/validation";
import { MIN_TOUCH, fontSizes, radii, spacing, useColors } from "../../theme";
import { Icon } from "../../ui";

type Props = { onSend: (body: string) => void; disabled?: boolean; placeholder?: string };

export function Composer({ onSend, disabled, placeholder = "Write a message" }: Props) {
  const colors = useColors();
  const [text, setText] = useState("");
  const canSend = text.trim().length > 0 && !disabled;

  function send() {
    if (!canSend) return;
    onSend(text.trim());
    setText("");
  }

  return (
    <View style={[styles.bar, { backgroundColor: colors.background, borderColor: colors.border }]}>
      <TextInput
        accessibilityLabel="Message"
        editable={!disabled}
        maxLength={MESSAGE_MAX}
        multiline
        onChangeText={setText}
        placeholder={placeholder}
        placeholderTextColor={colors.textMuted}
        style={[styles.input, { backgroundColor: colors.surface, borderColor: colors.border, color: colors.text }]}
        value={text}
      />
      <Pressable
        accessibilityLabel="Send message"
        accessibilityRole="button"
        accessibilityState={{ disabled: !canSend }}
        disabled={!canSend}
        onPress={send}
        style={[styles.send, { backgroundColor: canSend ? colors.primary : colors.border }]}
      >
        <Icon color={canSend ? colors.onPrimary : colors.textMuted} name="send" size={20} />
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    minHeight: MIN_TOUCH,
    maxHeight: 120,
    borderRadius: radii.lg,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    paddingTop: 10,
    paddingBottom: 10,
    fontSize: fontSizes.body
  },
  send: {
    width: MIN_TOUCH,
    height: MIN_TOUCH,
    borderRadius: MIN_TOUCH / 2,
    alignItems: "center",
    justifyContent: "center"
  }
});
