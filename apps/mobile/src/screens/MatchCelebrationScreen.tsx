import React, { useEffect, useRef } from "react";
import { Animated, StyleSheet, View } from "react-native";
import { BRAND, radius, spacing, useTheme } from "../theme";
import { useMe } from "../hooks/useAuth";
import type { MainScreenProps } from "../navigation/types";
import { ScreenContainer } from "../shared/layout";
import { Avatar, Button, Text } from "../shared/ui";

export default function MatchCelebrationScreen({ navigation, route }: MainScreenProps<"MatchCelebration">) {
  const theme = useTheme();
  const me = useMe().data;
  const { match } = route.params;
  const scale = useRef(new Animated.Value(0.6)).current;
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.parallel([
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }),
      Animated.timing(opacity, { toValue: 1, duration: 300, useNativeDriver: true })
    ]).start();
  }, [opacity, scale]);

  const sendMessage = () =>
    navigation.replace("Chat", {
      conversationId: match.conversationId,
      matchId: match.matchId,
      userId: match.user.userId,
      name: match.user.firstName
    });

  return (
    <ScreenContainer contentStyle={[styles.content, { backgroundColor: theme.background }]}>
      <Animated.View style={[styles.center, { opacity, transform: [{ scale }] }]}>
        <Text center tone="primary" variant="display">
          It's a match
        </Text>
        <Text center tone="muted">
          You and {match.user.firstName} liked each other.
        </Text>
        <View style={styles.photos}>
          <Avatar name={me?.profile?.firstName ?? "You"} photo={me?.profile?.photos[0]} size={120} />
          <View style={[styles.spark, { backgroundColor: theme.primary }]}>
            <Text style={{ color: theme.onPrimary }} variant="heading">
              ♥
            </Text>
          </View>
          <Avatar name={match.user.firstName} photo={match.user.photo} size={120} />
        </View>
        <Text center tone="muted" variant="caption">
          {BRAND.name} is free. Say hello whenever you are ready.
        </Text>
      </Animated.View>
      <View style={styles.actions}>
        <Button label="Send a message" onPress={sendMessage} />
        <Button label="Keep swiping" onPress={() => navigation.goBack()} variant="secondary" />
      </View>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  content: { flex: 1, justifyContent: "space-between", paddingVertical: spacing.xxl },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.lg },
  photos: { flexDirection: "row", alignItems: "center", gap: spacing.md, marginVertical: spacing.xl },
  spark: { width: 44, height: 44, borderRadius: radius.pill, alignItems: "center", justifyContent: "center" },
  actions: { gap: spacing.md }
});
