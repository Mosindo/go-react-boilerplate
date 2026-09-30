import React, { useState } from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Card, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import { moderationApi } from "../../lib/api/endpoints";
import type { ModerationReport } from "../../lib/api/types";
import { formatRelative, reportReasonLabels } from "../../lib/format";
import { showToast } from "../../lib/toast";

const KEY = ["moderation", "reports"] as const;

export default function ModerationScreen() {
  const client = useQueryClient();
  const query = useInfiniteQuery({
    queryKey: KEY,
    queryFn: ({ pageParam }) => moderationApi.reports(pageParam),
    initialPageParam: 0,
    getNextPageParam: (last) => last.nextOffset
  });
  const [busy, setBusy] = useState<string | null>(null);

  const act = async (report: ModerationReport, action: "dismiss" | "reviewed" | "suspend") => {
    setBusy(report.id);
    try {
      if (action === "suspend" && report.reportedUser) {
        await moderationApi.suspend(report.reportedUser.userId);
        showToast("Compte suspendu, signalements clôturés", "success");
      } else {
        await moderationApi.resolve(report.id, action === "dismiss" ? "dismissed" : "reviewed");
        showToast(action === "dismiss" ? "Signalement classé sans suite" : "Signalement traité", "success");
      }
      await client.invalidateQueries({ queryKey: KEY });
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setBusy(null);
    }
  };

  if (query.isLoading) return <LoadingState />;
  if (query.error) return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  const reports = query.data?.pages.flatMap((p) => p.reports) ?? [];

  return (
    <Screen edges={["bottom"]} testID="moderation-screen">
      <FlatList
        ListEmptyComponent={<EmptyState icon="checkmark-done-outline" message="Aucun signalement en attente." title="Tout est traité" />}
        contentContainerStyle={styles.list}
        data={reports}
        keyExtractor={(r) => r.id}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        renderItem={({ item }) => <ReportCard busy={busy === item.id} onAction={(a) => void act(item, a)} report={item} />}
      />
    </Screen>
  );
}

function ReportCard({ report, busy, onAction }: { report: ModerationReport; busy: boolean; onAction: (a: "dismiss" | "reviewed" | "suspend") => void }) {
  const { colors } = useTheme();
  const user = report.reportedUser;
  return (
    <Card style={styles.card}>
      <View style={styles.header}>
        <Avatar name={user?.firstName || "?"} size={48} uri={user?.photo?.url} />
        <View style={styles.flex}>
          <Text variant="label">{user ? `${user.firstName || "Membre"}${user.age ? `, ${user.age}` : ""}` : "Compte supprimé"}</Text>
          <Text tone="muted" variant="caption">
            {reportReasonLabels[report.reason]} · {formatRelative(report.createdAt)}
          </Text>
        </View>
        {report.openReportsOnUser > 1 ? (
          <View style={[styles.badge, { backgroundColor: colors.dangerSoft }]}>
            <Text tone="danger" variant="overline">
              {report.openReportsOnUser} SIGNALEMENTS
            </Text>
          </View>
        ) : null}
      </View>
      {report.details ? <Text>« {report.details} »</Text> : null}
      {report.reportedProfile?.bio ? (
        <Text tone="muted" variant="caption">
          Bio : {report.reportedProfile.bio}
        </Text>
      ) : null}
      <View style={styles.actions}>
        <Button disabled={busy} label="Sans suite" onPress={() => onAction("dismiss")} size="sm" variant="ghost" />
        <Button disabled={busy} label="Traité" onPress={() => onAction("reviewed")} size="sm" variant="secondary" />
        {user && !report.reportedUserSuspended ? (
          <Button disabled={busy} label="Suspendre" onPress={() => onAction("suspend")} size="sm" testID="moderation-suspend" variant="danger" />
        ) : null}
      </View>
    </Card>
  );
}

const styles = StyleSheet.create({
  list: { padding: 16, gap: 12, flexGrow: 1 },
  card: { padding: 16, gap: 10 },
  header: { flexDirection: "row", alignItems: "center", gap: 12 },
  flex: { flex: 1, gap: 2 },
  badge: { paddingHorizontal: 8, paddingVertical: 4, borderRadius: 8 },
  actions: { flexDirection: "row", justifyContent: "flex-end", flexWrap: "wrap", gap: 8 }
});
