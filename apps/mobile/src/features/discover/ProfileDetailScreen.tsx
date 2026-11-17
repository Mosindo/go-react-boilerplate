import React from "react";
import { StyleSheet, View, useWindowDimensions } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { useQuery } from "@tanstack/react-query";
import { errorMessage, mediaUrl } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { RootScreenProps } from "../../navigation/types";
import { colors, layout, radii, spacing } from "../../theme";
import { Button, Chip, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { GENDER_LABELS, distanceLabel } from "../../utils/format";
import { useBlockUser } from "../safety/useSafety";

export function ProfileDetailScreen({ navigation, route }: RootScreenProps<"ProfileDetail">) {
  const { width } = useWindowDimensions();
  const photoWidth = Math.min(width, layout.maxContentWidth) - spacing.lg * 2;
  const profile = useQuery({ queryKey: queryKeys.publicProfile(route.params.userId), queryFn: () => profileApi.view(route.params.userId) });
  const block = useBlockUser(() => navigation.popToTop());

  if (profile.isLoading) {
    return (
      <Screen edges={["left", "right"]}>
        <LoadingState />
      </Screen>
    );
  }
  if (profile.isError || !profile.data) {
    return (
      <Screen edges={["left", "right"]}>
        <ErrorState message={errorMessage(profile.error, "Ce profil n'est plus disponible.")} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  const p = profile.data;
  const distance = distanceLabel(p.distanceKm);

  return (
    <Screen scroll edges={["left", "right", "bottom"]} testID="profile-detail">
      <View style={styles.header}>
        <Text variant="display">
          {p.firstName}
          <Text variant="display" tone="muted" style={styles.age}>
            {"  "}
            {p.age}
          </Text>
        </Text>
        {p.isDemo ? (
          <Text variant="caption" tone="gold">
            Profil de démonstration
          </Text>
        ) : null}
        <View style={styles.facts}>
          <Fact icon="person-outline" text={GENDER_LABELS[p.gender]} />
          {p.jobTitle ? <Fact icon="briefcase-outline" text={p.jobTitle} /> : null}
          {distance ? <Fact icon="location-outline" text={`${distance}${p.city ? ` · ${p.city}` : ""}`} /> : null}
        </View>
      </View>

      {p.photos.map((photo, i) => (
        <React.Fragment key={photo.id}>
          <Image
            source={{ uri: mediaUrl(photo.url) }}
            style={[styles.photo, { width: photoWidth, height: (photoWidth * 5) / 4 }]}
            contentFit="cover"
            accessibilityLabel={`Photo ${i + 1} de ${p.firstName}`}
          />
          {i === 0 && p.bio ? (
            <View style={styles.block}>
              <Text variant="overline" tone="subtle">
                À propos
              </Text>
              <Text>{p.bio}</Text>
            </View>
          ) : null}
          {i === 0 && p.interests.length > 0 ? (
            <View style={styles.block}>
              <Text variant="overline" tone="subtle">
                Centres d’intérêt
              </Text>
              <View style={styles.chips}>
                {p.interests.map((interest) => (
                  <Chip key={interest.id} label={interest.label} />
                ))}
              </View>
            </View>
          ) : null}
        </React.Fragment>
      ))}

      <View style={styles.safety}>
        <Button
          label={`Signaler ${p.firstName}`}
          variant="secondary"
          testID="profile-report"
          icon={<Ionicons name="flag-outline" size={18} color={colors.text} />}
          onPress={() => navigation.navigate("Report", { userId: p.userId, firstName: p.firstName })}
        />
        <Button
          label={`Bloquer ${p.firstName}`}
          variant="danger"
          testID="profile-block"
          loading={block.isPending}
          onPress={() => block.confirm(p.userId, p.firstName)}
        />
        <Text variant="caption" tone="subtle" center>
          {p.firstName} ne saura pas que tu l’as bloqué·e ou signalé·e.
        </Text>
      </View>
    </Screen>
  );
}

function Fact({ icon, text }: { icon: React.ComponentProps<typeof Ionicons>["name"]; text: string }) {
  return (
    <View style={styles.fact}>
      <Ionicons name={icon} size={16} color={colors.textMuted} />
      <Text tone="muted" variant="label">
        {text}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  header: {
    gap: spacing.sm,
    marginBottom: spacing.lg
  },
  age: {
    fontWeight: "400"
  },
  facts: {
    gap: spacing.xs
  },
  fact: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  photo: {
    borderRadius: radii.lg,
    marginBottom: spacing.lg,
    backgroundColor: colors.surface
  },
  block: {
    gap: spacing.sm,
    marginBottom: spacing.xl
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  },
  safety: {
    gap: spacing.md,
    marginTop: spacing.lg
  }
});
