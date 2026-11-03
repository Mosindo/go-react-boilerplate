import React, { useState } from "react";
import { FlatList, StyleSheet, Text, View, useWindowDimensions } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { errorMessage } from "../api/client";
import { ReportModal } from "../features/safety/ReportModal";
import { confirmAction, notify } from "../lib/dialogs";
import { distanceLabel } from "../lib/format";
import type { MainStackParams } from "../navigation/types";
import { spacing, useTheme } from "../theme";
import { AuthImage } from "../ui/AuthImage";
import { Body, Button, Chip, ChipRow, Screen, Title } from "../ui/components";

type Props = NativeStackScreenProps<MainStackParams, "UserProfile">;

export function UserProfileScreen({ route, navigation }: Props) {
  const { user } = route.params;
  const theme = useTheme();
  const queryClient = useQueryClient();
  const { width } = useWindowDimensions();
  const [reporting, setReporting] = useState(false);
  const photoWidth = Math.min(width, 560);
  const distance = distanceLabel(user.distanceKm);

  const block = () =>
    confirmAction(`Bloquer ${user.firstName} ?`, "Vous ne verrez plus ce profil et il ne verra plus le vôtre.", "Bloquer", () => {
      api
        .block(user.id)
        .then(() =>
          Promise.all([
            queryClient.invalidateQueries({ queryKey: ["conversations"] }),
            queryClient.invalidateQueries({ queryKey: ["likes"] })
          ])
        )
        .then(() => navigation.popToTop())
        .catch((e) => notify("Erreur", errorMessage(e)));
    });

  return (
    <Screen padded={false}>
      <FlatList
        horizontal
        pagingEnabled
        data={user.photos}
        keyExtractor={(p) => p.id}
        showsHorizontalScrollIndicator={false}
        renderItem={({ item, index }) => (
          <AuthImage
            path={item.url}
            label={`Photo ${index + 1} de ${user.firstName}`}
            style={{ width: photoWidth, height: photoWidth * 1.25 }}
          />
        )}
        style={{ flexGrow: 0 }}
      />
      <View style={styles.content}>
        <Title>{`${user.firstName}, ${user.age}`}</Title>
        {user.city || distance ? <Body muted>{[user.city, distance].filter(Boolean).join(" · ")}</Body> : null}
        {user.bio ? <Body>{user.bio}</Body> : null}
        {user.interests.length > 0 ? (
          <View style={{ gap: spacing.sm }}>
            <Text style={{ color: theme.textMuted, fontWeight: "700" }}>Centres d&apos;intérêt</Text>
            <ChipRow>
              {user.interests.map((interest) => (
                <Chip key={interest.slug} label={interest.label} />
              ))}
            </ChipRow>
          </View>
        ) : null}
        <Button title="Signaler" variant="secondary" onPress={() => setReporting(true)} testID="report-user" />
        <Button title="Bloquer" variant="danger" onPress={block} testID="block-user" />
      </View>
      <ReportModal
        userId={reporting ? user.id : null}
        name={user.firstName}
        onClose={(sent) => {
          setReporting(false);
          if (sent) {
            notify("Merci", "Votre signalement a été envoyé.");
          }
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({ content: { padding: spacing.xl, gap: spacing.md } });
