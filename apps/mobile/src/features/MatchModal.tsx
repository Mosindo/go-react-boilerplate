import React, { useEffect, useRef } from "react";
import { Animated, Modal, View } from "react-native";
import { t } from "../i18n/fr";
import { createStyles } from "../theme/ThemeProvider";
import { Button, Photo, Text } from "../ui";
import { discoverStore, useDiscoverState } from "./discoverStore";

const useStyles = createStyles(({ colors, spacing, radius, contentMaxWidth }) => ({
  backdrop: {
    flex: 1,
    backgroundColor: colors.overlay,
    alignItems: "center",
    justifyContent: "center",
    padding: spacing.lg
  },
  card: {
    width: "100%",
    maxWidth: Math.min(contentMaxWidth, 420),
    backgroundColor: colors.surface,
    borderRadius: radius.lg,
    padding: spacing.xl,
    gap: spacing.lg,
    alignItems: "center"
  },
  photo: {
    width: 132,
    height: 132,
    borderRadius: 66,
    borderWidth: 4,
    borderColor: colors.highlight
  },
  actions: { alignSelf: "stretch", gap: spacing.sm }
}));

/** "It's a match" celebration shown after a mutual like. */
export function MatchModal({
  onSendMessage
}: {
  onSendMessage: (
    matchId: string,
    userId: string,
    firstName: string,
    thumbUrl: string | null
  ) => void;
}) {
  const styles = useStyles();
  const { celebration } = useDiscoverState();
  const scale = useRef(new Animated.Value(0.8)).current;
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!celebration) return;
    scale.setValue(0.8);
    opacity.setValue(0);
    Animated.parallel([
      Animated.spring(scale, { toValue: 1, useNativeDriver: true, bounciness: 10 }),
      Animated.timing(opacity, { toValue: 1, duration: 200, useNativeDriver: true })
    ]).start();
  }, [celebration, scale, opacity]);

  const close = () => discoverStore.celebrate(null);
  const name = celebration?.match.user.firstName ?? "";

  return (
    <Modal animationType="fade" onRequestClose={close} transparent visible={!!celebration}>
      <View style={styles.backdrop}>
        <Animated.View
          accessibilityViewIsModal
          style={[styles.card, { opacity, transform: [{ scale }] }]}
          testID="match-modal"
        >
          <Photo
            label={t.a11y.photoOf(name)}
            name={name}
            path={celebration?.card?.photos[0]?.url ?? celebration?.match.user.thumbUrl ?? null}
            style={styles.photo}
          />
          <View>
            <Text center tone="accent" variant="title">
              {t.discover.matchTitle}
            </Text>
            <Text center tone="muted">
              {t.discover.matchText(name)}
            </Text>
          </View>
          <View style={styles.actions}>
            <Button
              label={t.discover.sendMessage}
              onPress={() => {
                if (!celebration) return;
                const { match } = celebration;
                close();
                onSendMessage(
                  match.id,
                  match.user.userId,
                  match.user.firstName,
                  match.user.thumbUrl
                );
              }}
              testID="match-send-message"
            />
            <Button
              label={t.discover.keepSwiping}
              onPress={close}
              testID="match-dismiss"
              variant="ghost"
            />
          </View>
        </Animated.View>
      </View>
    </Modal>
  );
}
