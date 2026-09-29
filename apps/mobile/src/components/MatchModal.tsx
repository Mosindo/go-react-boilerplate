import React, { useEffect, useState } from "react";
import { Animated, Image, Modal, StyleSheet, View } from "react-native";
import type { ConversationSummary, Photo } from "../api/models";
import { useTheme } from "../shared/ui/theme";
import { AppButton, Txt, useReducedMotion } from "./kit";
import { photoUri } from "./photoUrl";

type Props = {
  visible: boolean;
  conversation: ConversationSummary | null;
  /** Photo of the current user, when known. */
  myPhoto?: Photo | null;
  onSayHello: (conversationId: string) => void;
  onKeepSwiping: () => void;
};

const SIZE = 132;

function Portrait({
  photo,
  name,
  tilt
}: {
  photo: Photo | null | undefined;
  name: string;
  tilt: string;
}) {
  const { colors, spacing } = useTheme();
  const uri = photoUri(photo);
  return (
    <View
      style={[
        styles.portrait,
        {
          borderColor: colors.background,
          backgroundColor: colors.primarySoft,
          transform: [{ rotate: tilt }],
          marginHorizontal: -spacing.md
        }
      ]}
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
    >
      {uri ? (
        <Image source={{ uri }} resizeMode="cover" style={styles.portraitImage} />
      ) : (
        <Txt variant="title" tone="primary">
          {name.charAt(0).toUpperCase()}
        </Txt>
      )}
    </View>
  );
}

export function MatchModal({ visible, conversation, myPhoto, onSayHello, onKeepSwiping }: Props) {
  const { colors, spacing } = useTheme();
  const reduced = useReducedMotion();
  const [scale] = useState(() => new Animated.Value(0.8));
  const [fade] = useState(() => new Animated.Value(0));

  useEffect(() => {
    if (!visible) return;
    if (reduced) {
      scale.setValue(1);
      fade.setValue(1);
      return;
    }
    scale.setValue(0.8);
    fade.setValue(0);
    Animated.parallel([
      Animated.spring(scale, {
        toValue: 1,
        friction: 5,
        useNativeDriver: true
      }),
      Animated.timing(fade, {
        toValue: 1,
        duration: 220,
        useNativeDriver: true
      })
    ]).start();
  }, [visible, reduced, scale, fade]);

  if (!conversation) return null;
  const name = conversation.user.firstName;

  return (
    <Modal
      visible={visible}
      transparent
      animationType="fade"
      onRequestClose={onKeepSwiping}
      statusBarTranslucent
    >
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Animated.View
          testID="match-modal"
          accessibilityViewIsModal
          accessibilityLiveRegion="polite"
          style={[
            styles.card,
            {
              backgroundColor: colors.background,
              borderColor: colors.border,
              padding: spacing.xl,
              gap: spacing.lg,
              opacity: fade,
              transform: [{ scale }]
            }
          ]}
        >
          <Txt variant="title" tone="primary" style={styles.center} accessibilityRole="header">
            It's a match!
          </Txt>
          <View style={styles.portraits}>
            <Portrait photo={myPhoto} name="Me" tilt="-6deg" />
            <Portrait photo={conversation.user.photo} name={name} tilt="6deg" />
          </View>
          <Txt tone="muted" style={styles.center}>
            {`You and ${name} liked each other.`}
          </Txt>
          <View style={{ gap: spacing.sm }}>
            <AppButton
              label="Say hello"
              onPress={() => onSayHello(conversation.id)}
              testID="match-say-hello"
              fullWidth
            />
            <AppButton
              label="Keep swiping"
              variant="ghost"
              onPress={onKeepSwiping}
              testID="match-keep-swiping"
              fullWidth
            />
          </View>
        </Animated.View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    padding: 24
  },
  card: {
    width: "100%",
    maxWidth: 380,
    borderRadius: 28,
    borderWidth: StyleSheet.hairlineWidth
  },
  center: { textAlign: "center" },
  portraits: {
    flexDirection: "row",
    justifyContent: "center",
    paddingVertical: 8
  },
  portrait: {
    width: SIZE,
    height: SIZE,
    borderRadius: SIZE / 2,
    borderWidth: 4,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  portraitImage: { width: "100%", height: "100%" }
});
