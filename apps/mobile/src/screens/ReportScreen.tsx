import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { reportUser } from "../api/social";
import type { ReportReason } from "../api/types";
import { Button, Chip, Screen, Text, TextField, useToast } from "../components";
import { errorMessage } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { spacing } from "../theme";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Fake profile" },
  { value: "harassment", label: "Harassment" },
  { value: "inappropriate_content", label: "Inappropriate content" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Seems under 18" },
  { value: "other", label: "Something else" }
];

export function ReportScreen({ navigation, route }: NativeStackScreenProps<MainStackParamList, "Report">) {
  const { userId, name, conversationId } = route.params;
  const toast = useToast();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const mutation = useMutation({
    mutationFn: () =>
      reportUser({ userId, reason: reason as ReportReason, details: details.trim() || undefined, conversationId }),
    onSuccess: () => {
      toast.show("Thanks. Our team will review your report.", "success");
      navigation.goBack();
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  return (
    <Screen scroll testID="report-screen">
      <View style={{ gap: spacing.lg }}>
        <Text variant="heading">Report {name}</Text>
        <Text tone="muted">Reports are confidential. You can also block this person from their profile.</Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {REASONS.map((r) => (
            <Chip key={r.value} label={r.label} selected={reason === r.value} onPress={() => setReason(r.value)} testID={`report-reason-${r.value}`} />
          ))}
        </View>
        <TextField label="Details (optional)" value={details} onChangeText={setDetails} multiline maxLength={1000} />
        <Button title="Send report" onPress={() => mutation.mutate()} disabled={!reason} loading={mutation.isPending} testID="report-submit" />
      </View>
    </Screen>
  );
}
