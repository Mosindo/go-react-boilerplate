import React from 'react';
import {
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  View,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { fr } from '../../i18n/fr';
import { MAX_CONTENT_WIDTH, radius, spacing, useTheme } from '../../theme';
import { Button } from './Button';
import { Text } from './Text';

/** Bottom-aligned modal shell, constrained on wide screens. */
export function Sheet({
  visible,
  onClose,
  children,
  testID,
  title,
}: {
  visible: boolean;
  onClose: () => void;
  children: React.ReactNode;
  testID?: string;
  title?: string;
}) {
  const { colors } = useTheme();
  return (
    <Modal
      visible={visible}
      transparent
      animationType="fade"
      onRequestClose={onClose}
      statusBarTranslucent
    >
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        style={[styles.backdrop, { backgroundColor: colors.scrim }]}
      >
        <Pressable
          style={StyleSheet.absoluteFill}
          accessibilityRole="button"
          accessibilityLabel={fr.common.close}
          onPress={onClose}
        />
        <View
          testID={testID}
          accessibilityViewIsModal
          style={[styles.sheet, { backgroundColor: colors.surface }]}
        >
          <ScrollView style={styles.scroll} keyboardShouldPersistTaps="handled" bounces={false}>
            {title ? (
              <Text variant="heading" style={styles.title} accessibilityRole="header">
                {title}
              </Text>
            ) : null}
            {children}
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

export type ActionItem = {
  key: string;
  label: string;
  icon?: keyof typeof Ionicons.glyphMap;
  destructive?: boolean;
  onPress: () => void;
};

export function ActionSheet({
  visible,
  onClose,
  items,
  title,
  testID = 'action-sheet',
}: {
  visible: boolean;
  onClose: () => void;
  items: ActionItem[];
  title?: string;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <Sheet visible={visible} onClose={onClose} title={title} testID={testID}>
      {items.map((item) => (
        <Pressable
          key={item.key}
          testID={`${testID}-${item.key}`}
          accessibilityRole="button"
          accessibilityLabel={item.label}
          onPress={() => {
            onClose();
            item.onPress();
          }}
          style={({ pressed }) => [
            styles.item,
            pressed ? { backgroundColor: colors.surfaceAlt } : null,
          ]}
        >
          {item.icon ? (
            <Ionicons
              name={item.icon}
              size={22}
              color={item.destructive ? colors.danger : colors.text}
              style={styles.itemIcon}
            />
          ) : null}
          <Text variant="body" tone={item.destructive ? 'danger' : 'default'}>
            {item.label}
          </Text>
        </Pressable>
      ))}
      <Button
        label={fr.common.cancel}
        variant="ghost"
        onPress={onClose}
        style={styles.cancel}
        testID={`${testID}-cancel`}
      />
    </Sheet>
  );
}

export function ConfirmDialog({
  visible,
  title,
  body,
  confirmLabel,
  destructive,
  loading,
  onConfirm,
  onCancel,
  children,
  confirmDisabled,
}: {
  visible: boolean;
  title: string;
  body?: string;
  confirmLabel: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  children?: React.ReactNode;
  confirmDisabled?: boolean;
}) {
  return (
    <Sheet visible={visible} onClose={onCancel} testID="confirm-dialog">
      <Text variant="title" style={styles.title} accessibilityRole="header">
        {title}
      </Text>
      {body ? (
        <Text tone="muted" style={styles.body}>
          {body}
        </Text>
      ) : null}
      {children}
      <Button
        label={confirmLabel}
        variant={destructive ? 'danger' : 'primary'}
        loading={loading}
        disabled={confirmDisabled}
        onPress={onConfirm}
        testID="confirm-dialog-confirm"
      />
      <Button
        label={fr.common.cancel}
        variant="ghost"
        onPress={onCancel}
        style={styles.cancel}
        testID="confirm-dialog-cancel"
      />
    </Sheet>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: 'flex-end', alignItems: 'center' },
  sheet: {
    width: '100%',
    maxWidth: MAX_CONTENT_WIDTH,
    maxHeight: '90%',
    borderTopLeftRadius: radius.lg,
    borderTopRightRadius: radius.lg,
    padding: spacing.lg,
    paddingBottom: spacing.xl,
  },
  scroll: { flexGrow: 0 },
  title: { marginBottom: spacing.md },
  body: { marginBottom: spacing.lg },
  item: {
    flexDirection: 'row',
    alignItems: 'center',
    minHeight: 52,
    paddingHorizontal: spacing.sm,
    borderRadius: radius.md,
  },
  itemIcon: { marginRight: spacing.md },
  cancel: { marginTop: spacing.sm },
});
