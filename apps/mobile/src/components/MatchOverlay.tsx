import React, { useEffect, useRef } from "react";
import { Animated, Modal, View } from "react-native";
import { Avatar, Button, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

type MatchOverlayProps = {
  visible: boolean;
  firstName: string;
  mePhotoUri?: string;
  theirPhotoUri?: string;
  myName?: string;
  onMessage: () => void;
  onContinue: () => void;
};

export function MatchOverlay({ firstName, myName, mePhotoUri, onContinue, onMessage, theirPhotoUri, visible }: MatchOverlayProps) {
  const palette = useTheme();
  const scale = useRef(new Animated.Value(0.85)).current;
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!visible) {
      scale.setValue(0.85);
      opacity.setValue(0);
      return;
    }
    Animated.parallel([
      Animated.spring(scale, { toValue: 1, friction: 6, useNativeDriver: true }),
      Animated.timing(opacity, { toValue: 1, duration: 200, useNativeDriver: true })
    ]).start();
  }, [opacity, scale, visible]);

  return (
    <Modal animationType="fade" onRequestClose={onContinue} statusBarTranslucent transparent visible={visible}>
      <View style={{ flex: 1, backgroundColor: palette.overlay, alignItems: "center", justifyContent: "center", padding: spacing.xl }}>
        <Animated.View
          accessibilityViewIsModal
          style={{
            width: "100%",
            maxWidth: 380,
            backgroundColor: palette.surface,
            borderRadius: radii.xl,
            padding: spacing.xl,
            alignItems: "center",
            gap: spacing.lg,
            opacity,
            transform: [{ scale }]
          }}
        >
          <Text accessibilityRole="header" tone="primary" variant="title" weight="bold">
            C&apos;est un match !
          </Text>
          <View style={{ flexDirection: "row", alignItems: "center" }}>
            <Avatar name={myName} ring size="xl" uri={mePhotoUri} />
            <Avatar name={firstName} ring size="xl" style={{ marginLeft: -spacing.lg }} uri={theirPhotoUri} />
          </View>
          <Text style={{ textAlign: "center" }} tone="muted">
            Vous et {firstName} vous êtes plu. Lancez la conversation !
          </Text>
          <View style={{ alignSelf: "stretch", gap: spacing.sm }}>
            <Button fullWidth label={`Écrire à ${firstName}`} onPress={onMessage} testID="match-send-message" />
            <Button fullWidth label="Continuer à découvrir" onPress={onContinue} testID="match-continue" variant="ghost" />
          </View>
        </Animated.View>
      </View>
    </Modal>
  );
}
