import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import { GENDER_LABELS } from "../api/types";
import { useInterests } from "../features/useInterests";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams } from "../navigation/types";
import { useTheme } from "../theme";
import { Button, Chip, ErrorState, LoadingState, PhotoImage, Screen, Text } from "../ui";

export function MeScreen() {
  const t = useTheme();
  const stack = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const interests = useInterests();
  const q = useQuery({ queryKey: queryKeys.overview, queryFn: profileApi.overview });
  if (q.isLoading) return <LoadingState />;
  const p = q.data?.profile;
  if (q.isError || !p)
    return <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />;
  const labels = new Map((interests.data ?? []).map((i) => [i.slug, i.label]));
  const hidden = q.data?.privacy?.isDiscoverable === false;

  return (
    <Screen scroll edges={["top"]}>
      <View style={styles.top}>
        <PhotoImage
          photoId={p.photos[0]?.id}
          style={[styles.avatar, { borderColor: t.primary }]}
          accessibilityLabel="Ma photo principale"
        />
        <Text variant="title">
          {p.firstName}, {p.age}
        </Text>
        <Text tone="muted">
          {GENDER_LABELS[p.gender]}
          {p.city ? ` · ${p.city}` : ""}
        </Text>
      </View>
      {hidden ? (
        <View style={[styles.banner, { backgroundColor: t.primarySoft }]}>
          <Ionicons name="eye-off-outline" size={18} color={t.primary} />
          <Text variant="caption" style={styles.flex}>
            Votre profil est en pause : vous n'apparaissez plus dans les découvertes.
          </Text>
        </View>
      ) : null}
      {p.bio ? <Text>{p.bio}</Text> : null}
      <View style={styles.tags}>
        {p.interests.map((slug) => (
          <Chip key={slug} label={labels.get(slug) ?? slug} />
        ))}
      </View>
      <Button
        label="Modifier mon profil et mes photos"
        onPress={() => stack.navigate("EditProfile")}
      />
      <Button
        label="Préférences de rencontre"
        variant="secondary"
        onPress={() => stack.navigate("Preferences")}
      />
      <Button
        label="Paramètres et confidentialité"
        variant="secondary"
        onPress={() => stack.navigate("Settings")}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  top: { alignItems: "center", gap: 6, paddingTop: 8 },
  avatar: { width: 128, height: 128, borderRadius: 64, borderWidth: 3, marginBottom: 6 },
  banner: { flexDirection: "row", alignItems: "center", gap: 10, padding: 12, borderRadius: 12 },
  flex: { flex: 1 },
  tags: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
