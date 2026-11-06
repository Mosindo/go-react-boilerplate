import React, { useState } from "react";
import { View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/errors";
import { safetyApi } from "../api/services";
import type { ReportReason } from "../api/types";
import { fr } from "../i18n/fr";
import { spacing } from "../theme";
import { Sheet } from "./Sheet";
import { useToast } from "./Toast";
import { Button, Chip, Input, Text } from "./ui";

const REASONS: ReportReason[] = [
  "spam",
  "fake",
  "harassment",
  "inappropriate",
  "underage",
  "other"
];

export function ReportSheet({
  visible,
  userId,
  onClose,
  onDone
}: {
  visible: boolean;
  userId: string;
  onClose: () => void;
  /** Called after a successful report; `blocked` is true when the user was also blocked. */
  onDone: (blocked: boolean) => void;
}) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!reason) return;
    setBusy(true);
    try {
      await safetyApi.report(userId, reason, details.trim(), alsoBlock);
      toast.show(fr.safety.reported, "success");
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      void queryClient.invalidateQueries({ queryKey: ["blocks"] });
      setReason(null);
      setDetails("");
      onDone(alsoBlock);
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Sheet visible={visible} onClose={onClose} title={fr.safety.reportTitle}>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {REASONS.map((r) => (
          <Chip
            key={r}
            testID={`reason-${r}`}
            label={fr.safety.reasons[r] ?? r}
            selected={reason === r}
            onPress={() => setReason(r)}
          />
        ))}
      </View>
      <Input
        label={fr.safety.details}
        value={details}
        onChangeText={setDetails}
        multiline
        maxLength={1000}
      />
      <Chip
        label={fr.safety.alsoBlock}
        selected={alsoBlock}
        onPress={() => setAlsoBlock((b) => !b)}
        testID="report-also-block"
      />
      <Button
        testID="report-submit"
        title={fr.safety.sendReport}
        disabled={!reason}
        loading={busy}
        onPress={() => void submit()}
      />
      <Button variant="ghost" title={fr.common.cancel} onPress={onClose} />
    </Sheet>
  );
}

export function ConfirmSheet({
  visible,
  title,
  message,
  confirmLabel,
  danger,
  busy,
  onConfirm,
  onClose,
  children
}: {
  visible: boolean;
  title: string;
  message?: string;
  confirmLabel: string;
  danger?: boolean;
  busy?: boolean;
  onConfirm: () => void;
  onClose: () => void;
  children?: React.ReactNode;
}) {
  return (
    <Sheet visible={visible} onClose={onClose} title={title}>
      {message ? <Text muted>{message}</Text> : null}
      {children}
      <Button
        testID="confirm-yes"
        title={confirmLabel}
        variant={danger ? "danger" : "primary"}
        loading={busy}
        onPress={onConfirm}
      />
      <Button testID="confirm-no" variant="ghost" title={fr.common.cancel} onPress={onClose} />
    </Sheet>
  );
}
