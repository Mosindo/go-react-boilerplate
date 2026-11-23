import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { isApiError } from "../../api/client";
import { safetyApi } from "../../api/dating";
import { genderLabels } from "../../api/types";
import { keys, usePublicProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { ConfirmDialog, EmptyView, ErrorView, LoadingView, showToast } from "../../shared/feedback";
import { Button, Chip, IconButton, Photo, Text, colors, spacing } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";
import { formatDistance } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "ProfileDetail">;

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId } = route.params;
  const query = usePublicProfile(userId);
  const client = useQueryClient();
  const { width } = useWindowDimensions();
  const [confirmBlock, setConfirmBlock] = useState(false);
  const photoWidth = Math.min(width, 640);

  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.matches });
      void client.invalidateQueries({ queryKey: keys.deck });
      showToast("Profil bloqué");
      navigation.popToTop();
    },
    onError: (e) => showToast(errorMessage(e), "danger")
  });

  if (query.isLoading) {
    return <LoadingView />;
  }
  if (query.isError || !query.data) {
    return (
      <SafeAreaView style={styles.safe}>
        {isApiError(query.error, "not_found") ? (
          <EmptyView
            actionLabel="Retour"
            message="Ce profil n'est plus disponible."
            onAction={() => navigation.goBack()}
            title="Profil introuvable"
          />
        ) : (
          <ErrorView onRetry={() => void query.refetch()} />
        )}
      </SafeAreaView>
    );
  }
  const p = query.data;
  const distance = formatDistance(p.distanceKm);

  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safe} testID="profile-detail">
      <ScrollView contentContainerStyle={styles.scroll}>
        <ScrollView horizontal pagingEnabled showsHorizontalScrollIndicator={false} style={{ width: photoWidth, alignSelf: "center" }}>
          {p.photos.map((photo, i) => (
            <Photo
              accessibilityLabel={`Photo ${i + 1} de ${p.firstName}`}
              key={photo.id}
              style={{ width: photoWidth, height: photoWidth * 1.2 }}
              url={photo.url}
            />
          ))}
        </ScrollView>
        <View style={styles.body}>
          <Text variant="title">
            {p.firstName},{" "}
            <Text variant="title" weight="regular">
              {p.age}
            </Text>
          </Text>
          <Text tone="muted">{[genderLabels[p.gender], p.city, distance].filter(Boolean).join(" · ")}</Text>
          {p.bio ? (
            <View style={styles.section}>
              <Text tone="muted" variant="eyebrow">
                À propos
              </Text>
              <Text>{p.bio}</Text>
            </View>
          ) : null}
          {p.interests.length > 0 ? (
            <View style={styles.section}>
              <Text tone="muted" variant="eyebrow">
                Centres d&apos;intérêt
              </Text>
              <View style={styles.chips}>
                {p.interests.map((i) => (
                  <Chip key={i.slug} label={i.label} />
                ))}
              </View>
            </View>
          ) : null}
          <View style={styles.safety}>
            <Button label={`Bloquer ${p.firstName}`} onPress={() => setConfirmBlock(true)} size="sm" variant="secondary" />
            <Button
              label={`Signaler ${p.firstName}`}
              onPress={() => navigation.navigate("Report", { userId, firstName: p.firstName })}
              size="sm"
              testID="profile-report"
              variant="danger"
            />
          </View>
        </View>
      </ScrollView>
      <IconButton accessibilityLabel="Fermer" glyph="✕" onPress={() => navigation.goBack()} style={styles.close} testID="profile-close" />
      <ConfirmDialog
        confirmLabel="Bloquer"
        destructive
        loading={block.isPending}
        message={`${p.firstName} ne pourra plus te voir ni t'écrire, et ne sera pas prévenu·e.`}
        onCancel={() => setConfirmBlock(false)}
        onConfirm={() => block.mutate()}
        title={`Bloquer ${p.firstName} ?`}
        visible={confirmBlock}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: colors.background
  },
  scroll: {
    paddingBottom: spacing.xxxl
  },
  body: {
    width: "100%",
    maxWidth: 640,
    alignSelf: "center",
    padding: spacing.xl,
    gap: spacing.sm
  },
  section: {
    marginTop: spacing.lg,
    gap: spacing.sm
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  },
  safety: {
    marginTop: spacing.xxl,
    gap: spacing.sm
  },
  close: {
    position: "absolute",
    top: spacing.xl,
    right: spacing.lg
  }
});
