import React from 'react';
import { Modal, StyleSheet, View } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { Ionicons } from '@expo/vector-icons';
import { fr } from '../i18n/fr';
import { openChat } from '../navigation/ref';
import { qk } from '../query/keys';
import { Avatar, Button, Text } from '../shared/ui';
import { matchCelebration } from '../store/appState';
import { MAX_CONTENT_WIDTH, radius, spacing, useTheme } from '../theme';
import type { MyProfile } from '../api/types';

/** Celebration shown when a swipe results in a match. */
export function MatchModal() {
  const { colors } = useTheme();
  const match = matchCelebration.use();
  const { data: mine } = useQuery<MyProfile>({ queryKey: qk.myProfile, enabled: false });
  const close = () => matchCelebration.set(null);
  return (
    <Modal
      visible={match !== null}
      transparent
      animationType="fade"
      onRequestClose={close}
      statusBarTranslucent
    >
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]}>
        {match ? (
          <View
            style={[styles.card, { backgroundColor: colors.surface }]}
            testID="match-modal"
            accessibilityViewIsModal
          >
            <View style={styles.avatars}>
              <Avatar name={mine?.firstName ?? '?'} path={mine?.photos[0]?.url} size={92} />
              <View style={[styles.heart, { backgroundColor: colors.primary }]}>
                <Ionicons name="heart" size={22} color={colors.onPrimary} />
              </View>
              <Avatar name={match.user.firstName} path={match.user.mainPhotoUrl} size={92} />
            </View>
            <Text
              variant="display"
              center
              accessibilityRole="header"
              testID="match-title"
              style={styles.title}
            >
              {fr.match.title}
            </Text>
            <Text tone="muted" center style={styles.body}>
              {fr.match.body(match.user.firstName)}
            </Text>
            <Button
              label={fr.match.send}
              icon="chatbubble-ellipses"
              testID="match-send"
              onPress={() => {
                const m = match;
                close();
                openChat(m.conversationId, m.user);
              }}
            />
            <Button
              label={fr.match.keep}
              variant="ghost"
              onPress={close}
              style={styles.keep}
              testID="match-keep"
            />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: spacing.lg },
  card: {
    width: '100%',
    maxWidth: MAX_CONTENT_WIDTH - 2 * spacing.lg,
    borderRadius: radius.lg,
    padding: spacing.xl,
  },
  avatars: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: spacing.xl,
  },
  heart: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
    marginHorizontal: -10,
    zIndex: 2,
  },
  title: { marginBottom: spacing.sm },
  body: { marginBottom: spacing.xl },
  keep: { marginTop: spacing.sm },
});
