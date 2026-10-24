import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { MainStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme/theme";
import { Avatar, Button, Icon, Screen, Text } from "../../ui";

type Props = NativeStackScreenProps<MainStackParamList, "MatchFound">;

export function MatchFoundScreen({ route, navigation }: Props) {
  const { colors } = useTheme();
  const { match } = route.params;
  const user = match.user;
  return (
    <Screen testID="match-found" contentStyle={styles.center}>
      <Icon name="heart" size={56} color={colors.accent} />
      <Text variant="display" center testID="match-title">
        C&apos;est un match !
      </Text>
      <Text center muted>
        Vous et {user.firstName} vous plaisez mutuellement.
      </Text>
      <View style={styles.avatar}>
        <Avatar photoId={user.photos[0]?.id} name={user.firstName} size={120} />
      </View>
      <View style={styles.actions}>
        <Button
          testID="match-send-message"
          title={`Écrire à ${user.firstName}`}
          onPress={() => {
            navigation.replace("Chat", {
              conversationId: match.conversationId,
              userId: user.userId,
              name: user.firstName,
              photoId: user.photos[0]?.id,
            });
          }}
        />
        <Button
          testID="match-continue"
          title="Continuer à découvrir"
          variant="ghost"
          onPress={() => navigation.goBack()}
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  center: { alignItems: "center", justifyContent: "center", gap: spacing.lg },
  avatar: { marginVertical: spacing.lg },
  actions: { alignSelf: "stretch", gap: spacing.sm },
});
