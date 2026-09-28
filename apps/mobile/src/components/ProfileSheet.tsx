import React, { useState } from "react";
import { Alert, Modal, ScrollView, StyleSheet, View } from "react-native";
import type { Candidate, ReportReason } from "../api/types";
import { errorMessage } from "../api/client";
import { formatDistance } from "../domain/distance";
import { useBlock, useReport } from "../hooks/useSafety";
import { showToast } from "../shared/feedback";
import { Button, Chip, IconButton, Text } from "../shared/ui";
import { spacing, useTheme } from "../theme";
import { PhotoCarousel } from "./PhotoCarousel";
import { ReportSheet } from "./ReportSheet";

export type ProfileSheetProps = {
  candidate: Candidate | null;
  myInterestIds: number[];
  onClose: () => void;
  /** Present when the sheet may take a decision (discover). */
  onDecide?: (action: "like" | "pass") => void;
  /** Called after the person was blocked so callers can drop them from their lists. */
  onBlocked?: (userId: string) => void;
};

export function ProfileSheet({ candidate, myInterestIds, onClose, onDecide, onBlocked }: ProfileSheetProps) {
  const theme = useTheme();
  const [photoIndex, setPhotoIndex] = useState(0);
  const [reporting, setReporting] = useState(false);
  const block = useBlock();
  const report = useReport();

  if (!candidate) {
    return null;
  }
  const distance = formatDistance(candidate.distanceKm);

  const askBlock = () => {
    Alert.alert(`Block ${candidate.firstName}?`, "You will no longer see each other anywhere in the app.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Block",
        style: "destructive",
        onPress: () =>
          block.mutate(candidate.userId, {
            onSuccess: () => {
              showToast(`${candidate.firstName} was blocked.`, "success");
              onBlocked?.(candidate.userId);
              onClose();
            },
            onError: (error) => showToast(errorMessage(error), "error")
          })
      }
    ]);
  };

  const submitReport = (reason: ReportReason, details: string) => {
    report.mutate(
      { userId: candidate.userId, reason, details: details || undefined },
      {
        onSuccess: () => {
          setReporting(false);
          showToast("Thanks. Your report was sent.", "success");
        },
        onError: (error) => showToast(errorMessage(error), "error")
      }
    );
  };

  const decide = (action: "like" | "pass") => {
    onClose();
    onDecide?.(action);
  };

  return (
    <Modal animationType="slide" onRequestClose={onClose} presentationStyle="pageSheet" visible>
      <View style={[styles.root, { backgroundColor: theme.background }]}>
        <ScrollView contentContainerStyle={styles.content}>
          <View style={styles.photo}>
            <PhotoCarousel
              index={photoIndex}
              name={candidate.firstName}
              onIndexChange={setPhotoIndex}
              photos={candidate.photos}
            />
          </View>
          <View style={styles.headerRow}>
            <View style={styles.flex}>
              <Text variant="title">
                {candidate.firstName}, {candidate.age}
              </Text>
              <Text tone="muted">{[candidate.locationLabel, distance].filter(Boolean).join(" · ")}</Text>
            </View>
            <IconButton glyph="✕" label="Close profile" onPress={onClose} />
          </View>
          {candidate.bio ? <Text>{candidate.bio}</Text> : null}
          {candidate.interests.length > 0 ? (
            <View style={styles.section}>
              <Text tone="muted" variant="label">
                Interests
              </Text>
              <View style={styles.chips}>
                {candidate.interests.map((interest) => (
                  <Chip highlighted={myInterestIds.includes(interest.id)} key={interest.id} label={interest.label} />
                ))}
              </View>
              {candidate.sharedInterestCount > 0 ? (
                <Text tone="muted" variant="caption">
                  Highlighted interests are ones you share.
                </Text>
              ) : null}
            </View>
          ) : null}
          <View style={styles.section}>
            <Button label="Report" onPress={() => setReporting(true)} variant="ghost" />
            <Button label="Block" onPress={askBlock} variant="ghost" />
          </View>
        </ScrollView>
        {onDecide ? (
          <View style={[styles.footer, { borderTopColor: theme.border }]}>
            <Button label="Pass" onPress={() => decide("pass")} style={styles.flex} variant="secondary" />
            <Button label="Like" onPress={() => decide("like")} style={styles.flex} />
          </View>
        ) : null}
        <ReportSheet
          name={candidate.firstName}
          onClose={() => setReporting(false)}
          onSubmit={submitReport}
          submitting={report.isPending}
          visible={reporting}
        />
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  flex: { flex: 1 },
  content: { padding: spacing.lg, gap: spacing.lg },
  photo: { height: 420 },
  headerRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  section: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  footer: { flexDirection: "row", gap: spacing.md, padding: spacing.lg, borderTopWidth: StyleSheet.hairlineWidth }
});
