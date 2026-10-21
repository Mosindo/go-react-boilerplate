import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import React from "react";
import { StyleSheet, View } from "react-native";
import { fetchProfile } from "../../api/profile";
import { keys } from "../../hooks/queries";
import type { RootStackParamList } from "../../navigation/types";
import { Avatar, Button, Text } from "../../shared/ui";
import { radius, spacing, useTheme } from "../../theme";

type Props = NativeStackScreenProps<RootStackParamList, "Match">;

/** "It's a match" overlay. */
export function MatchScreen({ navigation, route }: Props) {
  const theme = useTheme();
  const { match } = route.params;
  const me = useQuery({ queryKey: keys.profile, queryFn: fetchProfile });
  const myPhoto = me.data?.photos[0]?.thumbUrl;

  return (
    <View style={[styles.backdrop, { backgroundColor: theme.overlay }]} testID="match-screen">
      <View style={[styles.card, { backgroundColor: theme.surface }]}>
        <Text variant="title" tone="primary" center accessibilityRole="header">
          C'est un match !
        </Text>
        <Text tone="muted" center>
          Vous et {match.user.firstName} vous plaisez mutuellement.
        </Text>
        <View style={styles.avatars}>
          <Avatar name={me.data?.firstName ?? "Moi"} uri={myPhoto} size={92} />
          <Avatar name={match.user.firstName} uri={match.user.thumbUrl} size={92} />
        </View>
        <Button
          testID="match-send-message"
          label={`Écrire à ${match.user.firstName}`}
          onPress={() =>
            navigation.replace("Chat", { conversationId: match.conversationId, userId: match.user.id, name: match.user.firstName })
          }
        />
        <Button testID="match-continue" label="Continuer à découvrir" variant="ghost" onPress={() => navigation.goBack()} />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  card: { width: "100%", maxWidth: 420, borderRadius: radius.lg, padding: spacing.xl, gap: spacing.lg },
  avatars: { flexDirection: "row", justifyContent: "center", gap: spacing.lg }
});
