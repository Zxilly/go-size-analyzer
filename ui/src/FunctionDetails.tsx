import type { FunctionImpl } from "./tool/entry.ts";
import { functionPclnBytes } from "./tool/entry.ts";
import { formatBytes } from "./tool/utils.ts";

interface FunctionDetailsProps {
  entry: FunctionImpl;
  onClose: () => void;
}

export function FunctionDetails({ entry, onClose }: FunctionDetailsProps) {
  const fn = entry.getData();
  const evidence = fn.reference_evidence ?? [];
  const total = fn.reference_evidence_count ?? evidence.length;
  return (
    <section className="function-details" aria-label="Function details">
      <button type="button" className="function-details-back" onClick={onClose}>Back to file</button>
      <h2>{fn.full_name ?? entry.getName()}</h2>
      <p className="function-details-path">{entry.filePath}</p>
      <dl>
        <div>
          <dt>Machine code</dt>
          <dd>{formatBytes(fn.code_size)}</dd>
        </div>
        <div>
          <dt>Pcln metadata</dt>
          <dd>{formatBytes(functionPclnBytes(fn))}</dd>
        </div>
        <div>
          <dt>Referenced data</dt>
          <dd>
            {fn.referenced_bytes === undefined
              ? "Not measured"
              : formatBytes(fn.referenced_bytes)}
          </dd>
        </div>
        <div>
          <dt>Address</dt>
          <dd>{`0x${fn.addr.toString(16)}`}</dd>
        </div>
      </dl>
      <p>Referenced data may be shared by other functions. It is excluded from the area and does not measure potential savings.</p>
      <h3>Reference evidence</h3>
      {evidence.length > 0
        ? (
            <>
              <p>{`Showing ${evidence.length} of ${total} evidence records. File ranges use an exclusive end offset.`}</p>
              <div className="function-evidence-scroll">
                <table>
                  <thead>
                    <tr>
                      <th scope="col">File range</th>
                      <th scope="col">Section</th>
                      <th scope="col">Evidence</th>
                    </tr>
                  </thead>
                  <tbody>
                    {evidence.map(item => (
                      <tr key={`${item.offset}:${item.size}:${item.section}:${item.source}:${item.role}:${item.boundary}`}>
                        <td>
                          <code>{`0x${item.offset.toString(16)}–0x${(item.offset + item.size).toString(16)}`}</code>
                          <small>{formatBytes(item.size)}</small>
                        </td>
                        <td>{item.section || "Outside sections"}</td>
                        <td>
                          {item.source}
                          <small>{[item.role, item.boundary].filter(Boolean).join(" · ")}</small>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )
        : (
            <p>
              {fn.referenced_bytes === undefined
                ? "Reference analysis is unavailable in this report."
                : fn.referenced_bytes === 0
                  ? "No referenced file bytes were identified."
                  : "This report has no evidence samples. Generate it with --coverage-details to include them."}
            </p>
          )}
    </section>
  );
}
