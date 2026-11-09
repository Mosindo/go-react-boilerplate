import React, { useState } from "react";
import { FlatList, StyleSheet, View, useWindowDimensions } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi, safetyApi } from "../api/endpoints";
import { ReportSheet } from "../features/ReportSheet";
import { keys } from "../features/keys";
import { describeError } from "../lib/validation";
import { useNav } from "../nav/Navigation";
import { Chip, ActionSheet, ErrorState, LoadingView, PhotoImage, Screen, Text, useToast } from "../ui";
import { Pressable } from "react-native";

export default function ProfileDetailScreen({ userId }: { userId: string }) {
  const nav = useNav();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { width } = useWindowDimensions();
  const [menu, setMenu] = useState(false);
  const [report, setReport] = useState(false);
  const profile = useQuery({ queryKey: keys.publicProfile(userId), queryFn: () => profileApi.get(userId) });

  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      toast.show("Personne bloquée.");
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      nav.back();
    },
    onError: (e) => toast.show(describeError(e), "error")
  });

  const p = profile.data;
  return (
    <Screen
      title={p?.firstName ?? "Profil"}
      onBack={nav.back}
      right={
        p ? (
          <Pressable accessibilityRole="button" accessibilityLabel="Plus d'options" onPress={() => setMenu(true)} hitSlop={10}>
            <Text variant="heading">⋯</Text>
          </Pressable>
        ) : undefined
      }
      scroll
      contentStyle={styles.scroll}
    >
      {profile.isPending ? <LoadingView /> : null}
      {profile.isError ? (
        <ErrorState message={describeError(profile.error, "Ce profil n'est plus disponible.")} onRetry={() => void profile.refetch()} />
      ) : null}
      {p ? (
        <>
          <FlatList
            data={p.photos}
            keyExtractor={(photo) => photo.id}
            horizontal
            pagingEnabled
            showsHorizontalScrollIndicator={false}
            ListEmptyComponent={<PhotoImage name={p.firstName} style={{ width, height: width * 1.25 }} />}
            renderItem={({ item }) => <PhotoImage url={item.url} name={p.firstName} style={{ width, height: width * 1.25 }} />}
          />
          <View style={styles.info}>
            <Text variant="title">
              {p.firstName}, {p.age}
            </Text>
            <Text muted>{[p.city, p.distanceKm ? `à environ ${p.distanceKm} km` : null].filter(Boolean).join(" · ")}</Text>
            {p.bio ? <Text>{p.bio}</Text> : null}
            {p.interests.length > 0 ? (
              <View style={styles.wrap}>
                {p.interests.map((i) => (
                  <Chip key={i.id} label={i.label} />
                ))}
              </View>
            ) : null}
          </View>
          <ActionSheet
            visible={menu}
            actions={[
              { label: "Signaler", destructive: true, onPress: () => setReport(true) },
              { label: "Bloquer", destructive: true, onPress: () => block.mutate() }
            ]}
            onClose={() => setMenu(false)}
          />
          <ReportSheet
            visible={report}
            userId={userId}
            name={p.firstName}
            onClose={() => setReport(false)}
            onDone={() => {
              setReport(false);
              void queryClient.invalidateQueries({ queryKey: keys.discover });
              void queryClient.invalidateQueries({ queryKey: keys.conversations });
              nav.back();
            }}
          />
        </>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  scroll: { padding: 0, gap: 0, paddingBottom: 60 },
  info: { padding: 20, gap: 12 },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
