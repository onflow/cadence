/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import * as React from "react"
import {ReactElement} from "react"

export interface Data {
  [key: string]: Data;
}

interface TreeViewProps {
  data: Data,
  toggled?: boolean
  name?: string
  isChildElement?: boolean
  onOver?: (data: unknown) => boolean
  onLeave?: (data: unknown) => void
}

export const TreeView = ({
  data,
  toggled = false,
  name = null,
  isChildElement = false,
  onOver,
  onLeave,
}: TreeViewProps): ReactElement => {

  if (data === null) {
    return null
  }

  const [isToggled, setIsToggled] = React.useState(toggled);
  const isDataArray = Array.isArray(data);

  return (
    <div
      className={
        `tree-element
        ${toggled ? 'collapsed' : ''}
        ${isChildElement ? 'is-child' : ''}
      `}
      onMouseOver={(event) =>
        onOver && onOver(data) && event.stopPropagation()
      }
      onMouseLeave={() =>
        onLeave && onLeave(data)
      }
    >
      <span
        className={`toggler ${isToggled ? 'closed' : ''}`}
        onClick={() => setIsToggled(!isToggled)}
      />
      {name && <strong>{name}: </strong>}
      {isDataArray ? '[' : '{'}
      {isToggled && '...'}
      {Object.keys(data).map((v, i) => {
        const value = data[v]
        return typeof value === 'object' ? (
          <TreeView
            key={`${name}-${v}-${i}`}
            data={value}
            name={isDataArray ? null : v}
            isChildElement
            toggled={isToggled || toggled}
            onOver={onOver}
            onLeave={onLeave}
          />
        ) : (
          <p
            key={`${name}-${v}-${i}`}
            className={`tree-element ${isToggled ? 'collapsed' : ''}`}
          >
            {isDataArray ? '' : <strong>{v}: </strong>}
            {typeof value === 'string' ? `"${value}"` : value}
          </p>
        )
      })}
      {isDataArray ? ']' : '}'}
    </div>
  );
};
