import { message } from "../../test-utils";
import { createLocalId } from "../../realtime/cache";
import { buildThreadRows } from "./threadRows";

const at = (day: number, hour: number) => new Date(2026, 2, day, hour, 0).toISOString();
const now = new Date(2026, 2, 12, 18, 0);

describe("buildThreadRows", () => {
  it("adds a day separator after the oldest message of each day (newest-first order)", () => {
    const rows = buildThreadRows(
      [
        message({ id: "c", senderId: "other", createdAt: at(12, 10) }),
        message({ id: "b", senderId: "me", createdAt: at(11, 20) }),
        message({ id: "a", senderId: "other", createdAt: at(11, 9) })
      ],
      "me",
      false,
      now
    );
    expect(rows.map((row) => (row.kind === "day" ? row.label : row.message.id))).toEqual([
      "c",
      "Aujourd'hui",
      "b",
      "a",
      "Hier"
    ]);
  });

  it("omits the separator of the oldest loaded day while older pages remain", () => {
    const rows = buildThreadRows([message({ id: "a", createdAt: at(12, 9) })], "me", true, now);
    expect(rows.map((row) => row.kind)).toEqual(["message"]);
  });

  it("shows the receipt only under my latest message", () => {
    const statusOf = (rows: ReturnType<typeof buildThreadRows>) =>
      rows.flatMap((row) => (row.kind === "message" && row.status ? [`${row.message.id}:${row.status}`] : []));

    const read = buildThreadRows(
      [
        message({ id: "2", senderId: "me", readAt: at(12, 11), createdAt: at(12, 10) }),
        message({ id: "1", senderId: "me", createdAt: at(12, 9) })
      ],
      "me",
      false,
      now
    );
    expect(statusOf(read)).toEqual(["2:read"]);

    const sent = buildThreadRows(
      [
        message({ id: "2", senderId: "other", createdAt: at(12, 10) }),
        message({ id: "1", senderId: "me", createdAt: at(12, 9) })
      ],
      "me",
      false,
      now
    );
    expect(statusOf(sent)).toEqual(["1:sent"]);

    const sending = buildThreadRows(
      [message({ id: createLocalId(1), senderId: "me", createdAt: at(12, 10) })],
      "me",
      false,
      now
    );
    expect(statusOf(sending)).toEqual([`${createLocalId(1)}:sending`]);
  });
});
