import { useEffect, useRef } from "react";
import { Animated, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, PhotoImage, Text } from "../components/ui";
import type { RootStackParamList } from "../navigation/types";
import { radius, spacing, useColors } from "../theme";

type Props = NativeStackScreenProps<RootStackParamList, "MatchModal">;

export function MatchModalScreen({ navigation, route }: Props) {
  const { conversationId, userId, firstName, photoId } = route.params;
  const colors = useColors();
  const scale = useRef(new Animated.Value(0.8)).current;

  useEffect(() => {
    Animated.spring(scale, { toValue: 1, useNativeDriver: true }).start();
  }, [scale]);

  return (
    <View style={{ flex: 1, backgroundColor: colors.overlay, justifyContent: "center", padding: spacing.xl }}>
      <Animated.View
        accessibilityViewIsModal
        style={{
          transform: [{ scale }],
          backgroundColor: colors.surface,
          borderRadius: radius.lg,
          padding: spacing.xl,
          alignItems: "center",
        }}
      >
        <Text style={{ color: colors.primary, marginBottom: spacing.md }} variant="title">
          C&apos;est un match !
        </Text>
        {photoId ? (
          <PhotoImage
            photoId={photoId}
            style={{ width: 140, height: 140, borderRadius: 70, marginBottom: spacing.md }}
          />
        ) : null}
        <Text muted style={{ textAlign: "center", marginBottom: spacing.xl }}>
          Vous et {firstName} vous plaisez mutuellement.
        </Text>
        <Button
          onPress={() => {
            navigation.goBack();
            navigation.navigate("Chat", { conversationId, userId, name: firstName });
          }}
          style={{ alignSelf: "stretch" }}
          testID="match-message"
          title="Envoyer un message"
        />
        <Button onPress={() => navigation.goBack()} title="Continuer à découvrir" variant="ghost" />
      </Animated.View>
    </View>
  );
}
